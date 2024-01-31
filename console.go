package main

import (
	"os"
	"fmt"
	"time"
	"bufio"
	"strings"
)

type IPInfoStruct struct {
	TorrentUploaded map[string]int64
}
type PeerInfoStruct struct {
	Timestamp int64
	Port      map[int]bool
	Progress  float64
	Uploaded  int64
}
type PeerStatInfoStruct struct {
	Clients map[string]bool
}
type BlockPeerInfoStruct struct {
	Timestamp int64
	Port      int
}

var currentTimestamp int64 = 0
var lastQBURL = ""
var lastCleanTimestamp int64 = 0
var lastIPCleanTimestamp int64 = 0
var lastPeerCleanTimestamp int64 = 0
var lastPeerStatMapCleanTimestamp int64 = 0
var ipMap = make(map[string]IPInfoStruct)
var peerMap = make(map[string]PeerInfoStruct)
var peerMapStat = make(map[string]PeerStatInfoStruct)
var blockPeerMap = make(map[string]BlockPeerInfoStruct)

func AddIPInfo(clientIP string, torrentInfoHash string, clientUploaded int64) {
	if !config.IPUploadedCheck {
		return
	}
	var clientTorrentUploadedMap map[string]int64
	if info, exist := ipMap[clientIP]; !exist {
		clientTorrentUploadedMap = make(map[string]int64)
	} else {
		clientTorrentUploadedMap = info.TorrentUploaded
	}
	clientTorrentUploadedMap[torrentInfoHash] = clientUploaded
	ipMap[clientIP] = IPInfoStruct { TorrentUploaded: clientTorrentUploadedMap }
}
func AddPeerInfo(peerIP string, peerPort int, peerClient string, peerProgress float64, peerUploaded int64) {
	if config.MaxIPPortCount <= 0 && !config.BanByRelativeProgressUploaded {
		return
	}
	peerIP = strings.ToLower(peerIP)
	var peerPortMap map[int]bool
	if peer, exist := peerMap[peerIP]; !exist {
		peerPortMap = make(map[int]bool)
	} else {
		peerPortMap = peer.Port
	}
	var peerStatClients map[string]bool
	if peer, exist := peerMapStat[peerIP]; !exist {
		peerStatClients = make(map[string]bool)
	} else {
		peerStatClients = peer.Clients
	}
	peerStatClients[peerClient] = true
	peerPortMap[peerPort] = true
	peerMap[peerIP] = PeerInfoStruct { Timestamp: currentTimestamp, Port: peerPortMap, Progress: peerProgress, Uploaded: peerUploaded }
	peerMapStat[peerIP] = PeerStatInfoStruct { Clients: peerStatClients }
}
func AddBlockPeer(peerIP string, peerPort int) {
	blockPeerMap[strings.ToLower(peerIP)] = BlockPeerInfoStruct { Timestamp: currentTimestamp, Port: peerPort }
}
func IsBlockedPeer(peerIP string, peerPort int, updateTimestamp bool) bool {
	if blockPeer, exist := blockPeerMap[peerIP]; exist {
		if !useNewBanPeersMethod || blockPeer.Port < 0 || blockPeer.Port == peerPort {
			if updateTimestamp {
				blockPeer.Timestamp = currentTimestamp
			}
			return true
		}
	}
	return false
}
func IsIPTooHighUploaded(ipInfo IPInfoStruct, lastIPInfo IPInfoStruct) float64 {
	var totalUploaded int64 = 0
	for torrentInfoHash, torrentUploaded := range ipInfo.TorrentUploaded {
		totalUploaded += torrentUploaded
		if lastTorrentUploaded, exist := lastIPInfo.TorrentUploaded[torrentInfoHash]; exist {
			totalUploaded -= lastTorrentUploaded
		}
	}
	var totalUploadedMB float64 = (float64(totalUploaded) / 1024 / 1024)
	if totalUploadedMB > float64(config.IPUpCheckIncrementMB) {
		return totalUploadedMB
	}
	return 0
}
func IsProgressNotMatchUploaded(torrentTotalSize int64, clientProgress float64, clientUploaded int64) bool {
	if config.BanByProgressUploaded && torrentTotalSize > 0 && clientProgress >= 0 && clientUploaded > 0 {
		/*
		条件 1. 若客户端对 Peer 上传已大于等于 Torrnet 大小的 2%;
		条件 2. 但 Peer 报告进度乘以下载量再乘以一定防误判倍率, 却比客户端上传量还小;
		若满足以上条件, 则认为 Peer 是有问题的.
		e.g.:
		若 torrentTotalSize: 100GB, clientProgress: 1% (0.01), clientUploaded: 6GB, config.BanByPUStartPrecent: 2 (0.02), config.BanByPUAntiErrorRatio: 5;
		判断条件 1:
		torrentTotalSize * config.BanByPUStartPrecent = 100GB * 0.02 = 2GB, clientUploaded = 6GB >= 2GB
		满足此条件;
		判断条件 2:
		torrentTotalSize * clientProgress * config.BanByPUAntiErrorRatio = 100GB * 0.01 * 5 = 5GB, 5GB < clientUploaded = 6GB
		满足此条件;
		则该 Peer 将被封禁, 由于其报告进度为 1%, 算入 config.BanByPUAntiErrorRatio 滞后防误判倍率后为 5% (5GB), 但客户端实际却已上传 6GB.
		*/
		startUploaded := (float64(torrentTotalSize) * (float64(config.BanByPUStartPrecent) / 100))
		peerReportDownloaded := (float64(torrentTotalSize) * clientProgress)
		if (clientUploaded / 1024 / 1024) >= int64(config.BanByPUStartMB) && float64(clientUploaded) >= startUploaded && (peerReportDownloaded * float64(config.BanByPUAntiErrorRatio)) < float64(clientUploaded) {
			return true
		}
	}
	return false
}
func IsProgressNotMatchUploaded_Relative(peerInfo PeerInfoStruct, lastPeerInfo PeerInfoStruct) float64 {
	// 若客户端对 Peer 上传已大于 0, 且相对上传量大于起始上传量, 则继续判断.
	var relativeUploaded float64 = (float64(peerInfo.Uploaded - lastPeerInfo.Uploaded) / 1024 / 1024)
	if peerInfo.Uploaded > 0 && relativeUploaded > float64(config.BanByRelativePUStartMB) {
		relativeUploadedPrecent := (float64(lastPeerInfo.Uploaded) / float64(peerInfo.Uploaded))
		// 若相对上传百分比大于起始百分比, 则继续判断.
		if relativeUploadedPrecent > (float64(config.BanByRelativePUStartPrecent) / 100) {
			// 若相对上传百分比大于 Peer 报告进度乘以一定防误判倍率, 则认为 Peer 是有问题的.
			var peerReportProgress float64 = 0
			if peerInfo.Progress > 0 {
				peerReportProgress = (lastPeerInfo.Progress / peerInfo.Progress)
			}
			if relativeUploadedPrecent > (peerReportProgress * float64(config.BanByRelativePUAntiErrorRatio)) {
				return relativeUploaded
			}
		}
	}
	return 0
}
func ClearBlockPeer() int {
	cleanCount := 0
	if config.CleanInterval == 0 || (lastCleanTimestamp + int64(config.CleanInterval) < currentTimestamp) {
		for clientIP, clientInfo := range blockPeerMap {
			if currentTimestamp > (clientInfo.Timestamp + int64(config.BanTime)) {
				cleanCount++
				delete(blockPeerMap, clientIP)
			}
		}
		if cleanCount != 0 {
			lastCleanTimestamp = currentTimestamp
			Log("ClearBlockPeer", "已清理过期客户端: %d 个", true, cleanCount)
		}
	}
	return cleanCount
}
func CheckTorrent(torrentInfoHash string, torrentInfo TorrentStruct) (int, *TorrentPeersStruct) {
	Log("Debug-CheckTorrent", "%s", false, torrentInfoHash)
	if torrentInfoHash == "" {
		return -1, nil
	}
	if torrentInfo.NumLeechs < 1 {
		return -2, nil
	}
	torrentPeers := FetchTorrentPeers(torrentInfoHash)
	if torrentPeers == nil {
		return -3, nil
	}
	return 0, torrentPeers
}
func CheckPeer(peer PeerStruct, torrentInfoHash string, torrentTotalSize int64) int {
	if peer.IP == "" || peer.Client == "" || CheckPrivateIP(peer.IP) {
		return -1
	}
	if IsBlockedPeer(peer.IP, peer.Port, true) {
		Log("Debug-CheckPeer_IgnorePeer (Blocked)", "%s:%d %s", false, peer.IP, peer.Port, peer.Client)
		if peer.Port == -2 {
			return 4
		}
		if peer.Port == -1 {
			return 3
		}
		return 2
	}
	Log("Debug-CheckPeer", "%s %s", false, peer.IP, peer.Client)
	if IsProgressNotMatchUploaded(torrentTotalSize, peer.Progress, peer.Uploaded) {
		Log("CheckPeer_AddBlockPeer (Bad-Progress_Uploaded)", "%s:%d %s (TorrentTotalSize: %.2f MB, Progress: %.2f%%, Uploaded: %.2f MB)", true, peer.IP, peer.Port, peer.Client, (float64(torrentTotalSize) / 1024 / 1024), (peer.Progress * 100), (float64(peer.Uploaded) / 1024 / 1024))
		AddBlockPeer(peer.IP, peer.Port)
		return 1
	}
	for _, v := range blockListCompiled {
		if v.MatchString(peer.Client) {
			Log("CheckPeer_AddBlockPeer (Bad-Client)", "%s:%d %s", true, peer.IP, peer.Port, peer.Client)
			AddBlockPeer(peer.IP, peer.Port)
			return 1
		}
	}
	AddIPInfo(peer.IP, torrentInfoHash, peer.Uploaded)
	AddPeerInfo(peer.IP, peer.Port, peer.Client, peer.Progress, peer.Uploaded)
	return 0
}
func CheckAllIP(lastIPMap map[string]IPInfoStruct) int {
	if config.IPUploadedCheck && len(lastIPMap) > 0 && currentTimestamp > (lastIPCleanTimestamp + int64(config.IPUpCheckInterval)) {
		blockCount := 0
		for ip, ipInfo := range ipMap {
			if IsBlockedPeer(ip, -1, false) {
				continue
			}
			if lastIPInfo, exist := lastIPMap[ip]; exist {
				if uploadDuring := IsIPTooHighUploaded(ipInfo, lastIPInfo); uploadDuring > 0 {
					Log("CheckAllIP_AddBlockPeer (Too high uploaded)", "%s:%d (UploadDuring: %.2f MB)", true, ip, -1, uploadDuring)
					blockCount++
					AddBlockPeer(ip, -1)
				}
			}
		}
		lastIPCleanTimestamp = currentTimestamp
		ipMap = make(map[string]IPInfoStruct)
		return blockCount
	}
	return 0
}
func CheckAllPeer(lastPeerMap map[string]PeerInfoStruct) int {
	if (config.MaxIPPortCount > 0 || config.BanByRelativeProgressUploaded) && len(lastPeerMap) > 0 && currentTimestamp > (lastPeerCleanTimestamp + int64(config.PeerMapCleanInterval)) {
		blockCount := 0
		peerMapLoop:
		for ip, peerInfo := range peerMap {
			if IsBlockedPeer(ip, -1, false) || IsBlockedPeer(ip, -2, false) {
				continue
			}
			for port := range peerInfo.Port {
				if IsBlockedPeer(ip, port, false) {
					continue peerMapLoop
				}
			}
			if config.MaxIPPortCount > 0 {
				if len(peerInfo.Port) > int(config.MaxIPPortCount) {
					Log("CheckAllPeer_AddBlockPeer (Too many ports)", "%s:%d", true, ip, -1)
					AddBlockPeer(ip, -1)
					continue
				}
			}
			if config.BanByRelativeProgressUploaded {
				if lastPeerInfo, exist := lastPeerMap[ip]; exist {
					if uploadDuring := IsProgressNotMatchUploaded_Relative(peerInfo, lastPeerInfo); uploadDuring > 0 {
						Log("CheckAllPeer_AddBlockPeer (Bad-Relative_Progress_Uploaded)", "%s:%d (UploadDuring: %.2f MB)", true, ip, -2, uploadDuring)
						blockCount++
						AddBlockPeer(ip, -2)
					}
				}
			}
		}
		lastPeerCleanTimestamp = currentTimestamp
		peerMap = make(map[string]PeerInfoStruct)
		return blockCount
	}
	return 0
}
func GenClientStat() bool {
	if config.PeerStatMapCleanInterval > 0 && currentTimestamp > (lastPeerStatMapCleanTimestamp + int64(config.PeerStatMapCleanInterval)) {
		clientCountMap := map[string]int {}
		for _, peerStatInfo := range peerMapStat {
			for client, _ := range peerStatInfo.Clients {
				if clientCount, exist := clientCountMap[client]; exist {
					clientCountMap[client] = clientCount
				}
				clientCountMap[client] += 1
			}
		}
		if statFile, err := os.Create(GetStatFilePath()); err == nil {
			statFileWrite := bufio.NewWriter(statFile)
			statFileWrite.WriteString(fmt.Sprintf("客户端统计信息 (自 %s 至 %s)\n", GetDateTime(true, lastPeerStatMapCleanTimestamp), GetDateTime(true, 0)))
			for client, count := range clientCountMap {
				statFileWrite.WriteString(fmt.Sprintf("%s 的用户数: %d\n", client, count))
			}
			lastPeerStatMapCleanTimestamp = currentTimestamp
			peerMapStat = make(map[string]PeerStatInfoStruct)
			return true;
		}
	}
	return false;
}
func Task() {
	if config.QBURL == "" {
		Log("Task", "检测到 QBURL 为空, 可能是未配置且未能自动读取 qBittorrent 配置文件", false)
		return
	}
	if lastQBURL != config.QBURL {
		SubmitBlockPeer("")
		lastQBURL = config.QBURL
	}

	metadata := FetchMaindata()
	if metadata == nil {
		return
	}

	cleanCount := ClearBlockPeer()
	blockCount := 0
	ipBlockCount := 0
	emptyHashCount := 0
	noLeechersCount := 0
	badTorrentInfoCount := 0
	badPeersCount := 0
	lastIPMap := ipMap
	lastPeerMap := peerMap

	for torrentInfoHash, torrentInfo := range metadata.Torrents {
		torrentStatus, torrentPeers := CheckTorrent(torrentInfoHash, torrentInfo)
		switch torrentStatus {
			case -1:
				emptyHashCount++
			case -2:
				noLeechersCount++
			case -3:
				badTorrentInfoCount++
			case 0:
				for _, peers := range torrentPeers.Peers {
					peerStatus := CheckPeer(peers, torrentInfoHash, torrentInfo.TotalSize)
					switch peerStatus {
						case 3:
							ipBlockCount++
						case 1:
							blockCount++
						case -1:
							badPeersCount++
					}
				}
		}
		if config.SleepTime != 0 {
			time.Sleep(time.Duration(config.SleepTime) * time.Millisecond)
		}
	}

	currentIPBlockCount := CheckAllIP(lastIPMap)
	ipBlockCount += currentIPBlockCount
	blockCount += CheckAllPeer(lastPeerMap)
	GenClientStat()

	Log("Debug-Task_IgnoreEmptyHashCount", "%d", false, emptyHashCount)
	Log("Debug-Task_IgnoreNoLeechersCount", "%d", false, noLeechersCount)
	Log("Debug-Task_IgnoreBadTorrentInfoCount", "%d", false, badTorrentInfoCount)
	Log("Debug-Task_IgnoreBadPeersCount", "%d", false, badPeersCount)
	if cleanCount != 0 || blockCount != 0 {
		peersStr := GenBlockPeersStr()
		Log("Debug-Task_GenBlockPeersStr", "%s", false, peersStr)
		SubmitBlockPeer(peersStr)
		Log("Task", "此次封禁客户端: %d 个, 当前封禁客户端: %d 个, 此次封禁 IP 地址: %d 个, 当前封禁 IP 地址: %d 个", true, blockCount, len(blockPeerMap), currentIPBlockCount, ipBlockCount)
	}
}
func RunConsole() {
	RegFlag()
	if !LoadConfig(true) {
		Log("RunConsole", "读取配置文件失败或不完整", false)
		InitConfig()
	}
	if !Login() {
		Log("RunConsole", "认证失败", true)
		return
	}
	Log("RunConsole", "程序已启动", true)
	for range time.Tick(time.Duration(config.Interval) * time.Second) {
		currentTimestamp = time.Now().Unix()
		if !LoadConfig(false) {
			InitConfig()
		}
		Task()
	}
}
