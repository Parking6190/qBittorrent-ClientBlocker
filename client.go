package main

var currentClientType = ""

// 重复判断 nil 是因为输出的类型转换 (qB_MainDataStruct -> interface{}) 会导致 nil 比较失效.
func IsBanPort() bool {
	if qB_useNewBanPeersMethod {
		return true
	}

	return false
}
func IsSupportClient() bool {
	switch currentClientType {
		case "qBittorrent", "Transmission":
			return true
		default:
			return false
	}

	return false
}
func InitClient() {
	if currentClientType == "Transmission" {
		Tr_InitClient()
	}
}
func SetURLFromClient() {
	// 未设置的情况下, 应按内部客户端顺序逐个测试.
	if !qB_SetURL() {
		Tr_SetURL()
	}
}
func DetectClient() bool {
	if config.ClientType != "" {
		currentClientType = config.ClientType
		return true
	}

	currentClientType = "qBittorrent"
	if qB_GetAPIVersion() {
		Log("DetectClient", GetLangText("Success-DetectClient"), true, currentClientType)
		return true
	}

	currentClientType = "Transmission"
	if Tr_DetectVersion() {
		Log("DetectClient", GetLangText("Success-DetectClient"), true, currentClientType)
		return true
	}

	currentClientType = ""
	return false
}
func Login() bool {
	switch currentClientType {
		case "qBittorrent":
			return qB_Login()
		case "Transmission":
			return Tr_Login()
	}

	return false
}
func FetchTorrents() interface{} {
	switch currentClientType {
		case "qBittorrent":
			maindata := qB_FetchTorrents()
			if maindata == nil {
				return nil
			}
			return maindata
		case "Transmission":
			maindata := Tr_FetchTorrents()
			if maindata == nil {
				return nil
			}
			return maindata
	}

	return nil
}
func FetchTorrentPeers(infoHash string) interface{} {
	switch currentClientType {
		case "qBittorrent":
			torrentPeers := qB_FetchTorrentPeers(infoHash)
			if torrentPeers == nil {
				return nil
			}
			return torrentPeers
	}

	return nil
}
func SubmitBlockPeer(blockPeerMap map[string]BlockPeerInfoStruct) bool {
	switch currentClientType {
		case "qBittorrent":
			return qB_SubmitBlockPeer(blockPeerMap)
		case "Transmission":
			return Tr_SubmitBlockPeer(blockPeerMap)
	}

	return false
}
