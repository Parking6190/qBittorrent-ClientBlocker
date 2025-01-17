package main

import (
	"net/http"
	"strings"
	"io/ioutil"
)

func NewRequest(isPOST bool, url string, postdata string, withAuth bool, withHeader *map[string]string) *http.Request {
	var request *http.Request
	var err error

	if !isPOST {
		request, err = http.NewRequest("GET", url, nil)
	} else {
		request, err = http.NewRequest("POST", url, strings.NewReader(postdata))
	}

	if err != nil {
		Log("NewRequest", GetLangText("Error-NewRequest"), true, err.Error())
		return nil
	}

	request.Header.Set("User-Agent", programName + "/" + programVersion)

	setContentType := false

	if withHeader != nil {
		for k, v := range *withHeader {
			if strings.ToLower(k) == "content-type" {
				setContentType = true
			}
			request.Header.Set(k, v)
		}
	}

	if !setContentType && isPOST {
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	if currentClientType == "Transmission" && withAuth && Tr_csrfToken != "" {
		request.Header.Set("X-Transmission-Session-Id", Tr_csrfToken)
	}

	if withAuth && config.UseBasicAuth && config.ClientUsername != "" {
		request.SetBasicAuth(config.ClientUsername, config.ClientPassword)
	}

	return request
}
func Fetch(url string, tryLogin bool, withCookie bool, withHeader *map[string]string) (int, []byte) {
	request := NewRequest(false, url, "", withCookie, withHeader)
	if request == nil {
		return -1, nil
	}

	var response *http.Response
	var err error

	if withCookie {
		response, err = httpClient.Do(request)
	} else {
		response, err = httpClientWithoutCookie.Do(request)
	}

	if err != nil {
		Log("Fetch", GetLangText("Error-FetchResponse"), true, err.Error())
		return -2, nil
	}

	responseBody, err := ioutil.ReadAll(response.Body)
	defer response.Body.Close()

	if err != nil {
		Log("Fetch", GetLangText("Error-ReadResponse"), true, err.Error())
		return -3, nil
	}

	if response.StatusCode == 401 {
		Log("Fetch", GetLangText("Error-NoAuth"), true)
		return 401, nil
	}

	if response.StatusCode == 403 {
		if tryLogin {
			Login()
		}
		Log("Fetch", GetLangText("Error-Forbidden"), true)
		return 403, nil
	}

	if response.StatusCode == 409 {
		// 尝试获取并设置 CSRF Token.
		if currentClientType == "Transmission" {
			transmissionCSRFToken := response.Header.Get("X-Transmission-Session-Id")
			if transmissionCSRFToken != "" {
				Tr_SetCSRFToken(transmissionCSRFToken)
				return 409, nil
			}
		}

		if tryLogin {
			Login()
		}

		Log("Fetch", GetLangText("Error-Forbidden"), true)
		return 409, nil
	}

	if response.StatusCode == 404 {
		Log("Fetch", GetLangText("Error-NotFound"), true)
		return 404, nil
	}

	if response.StatusCode != 200 {
		Log("Fetch", GetLangText("Error-UnknownStatusCode"), true, response.StatusCode)
		return response.StatusCode, nil
	}

	return response.StatusCode, responseBody
}
func Submit(url string, postdata string, tryLogin bool, withCookie bool, withHeader *map[string]string) (int, []byte) {
	request := NewRequest(true, url, postdata, withCookie, withHeader)
	if request == nil {
		return -1, nil
	}

	var response *http.Response
	var err error

	if withCookie {
		response, err = httpClient.Do(request)
	} else {
		response, err = httpClientWithoutCookie.Do(request)
	}

	if err != nil {
		Log("Submit", GetLangText("Error-FetchResponse"), true, err.Error())
		return -2, nil
	}
	responseBody, err := ioutil.ReadAll(response.Body)
	defer response.Body.Close()

	if err != nil {
		Log("Submit", GetLangText("Error-ReadResponse"), true, err.Error())
		return -3, nil
	}

	if response.StatusCode == 401 {
		Log("Submit", GetLangText("Error-NoAuth"), true)
		return 401, nil
	}

	if response.StatusCode == 403 {
		if tryLogin {
			Login()
		}
		Log("Submit", GetLangText("Error-Forbidden"), true)
		return 403, nil
	}

	if response.StatusCode == 409 {
		// 尝试获取并设置 CSRF Token.
		if currentClientType == "Transmission" {
			transmissionCSRFToken := response.Header.Get("X-Transmission-Session-Id")
			if transmissionCSRFToken != "" {
				Tr_SetCSRFToken(transmissionCSRFToken)
				return 409, nil
			}
		}

		if tryLogin {
			Login()
		}

		Log("Fetch", GetLangText("Error-Forbidden"), true)
		return 409, nil
	}

	if response.StatusCode == 404 {
		Log("Submit", GetLangText("Error-NotFound"), true)
		return 404, nil
	}

	if response.StatusCode != 200 {
		Log("Submit", GetLangText("Error-UnknownStatusCode"), true, response.StatusCode)
		return response.StatusCode, nil
	}

	return response.StatusCode, responseBody
}
