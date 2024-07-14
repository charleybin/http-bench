package main

import (
    "bytes"
    "fmt"
    // "io/ioutil"
    "net/http"
    "sync"
    "sync/atomic"
    "time"
    "os"
    "io"
    "crypto/tls"
    "mime/multipart"
    "flag"
)

var (
    url         string
    numRequests int
    concurrency int
    filename    string
    successCount uint64
    failCount    uint64
)

func init() {
    flag.StringVar(&url, "url", "http://127.0.0.1:7862/ai/v1/voice2text", "POST请求目标URL")
    flag.IntVar(&numRequests, "numRequests", 2, "总请求数")
    flag.IntVar(&concurrency, "concurrency", 2, "并发请求数")
    flag.StringVar(&filename, "filename", "/media/Data2T/ai_code/gpt/repo/voice2text-server/voice2text-cpu-source/voice-10sec.m4a", "要上传的文件的路径")
}

// http-post-bench/http-post-bench --url="http://127.0.0.1:7862/ai/v1/voice2text" --filename="voice-180s.m4a" --numRequests=2 --concurrency=2

func main() {
    flag.Parse()

    var wg sync.WaitGroup
    start := time.Now()

    for i := 0; i < numRequests; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            sendPOSTRequest()
        }()
        if i >= concurrency {
            wg.Wait() // 等待当前并发请求完成，再继续发送新的请求
        }
    }

    wg.Wait() // 等待所有请求完成
    duration := time.Since(start)

    fmt.Printf("总请求数: %d\n", numRequests)
    fmt.Printf("总响应时间: %.2f ms\n", float64(duration.Nanoseconds())/1e6)
    fmt.Printf("成功请求数: %d\n", atomic.LoadUint64(&successCount))
    fmt.Printf("失败请求数: %d\n", atomic.LoadUint64(&failCount))
    fmt.Printf("平均响应时间: %.2f ms\n", float64(duration.Nanoseconds())/float64(numRequests)/1e6)
}

func FileUploadRequest(uri string, path string) (*http.Request, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", path)
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(part, file)

	err = writer.Close()
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequest("POST", uri, body)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Close = true

	return request, err
}

func sendPOSTRequest() {
    req, err := FileUploadRequest(url, filename) 
    if err != nil {
        atomic.AddUint64(&failCount, 1)
        fmt.Println("创建请求失败:", err)
        return
    }
    
    // 构造HTTP 对象
    tr := &http.Transport{
    	TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // 忽略证书验证
    }
    client := &http.Client{Transport: tr}
    //client := new(http.Client)
    resp, err := client.Do(req)
    if err != nil {
        atomic.AddUint64(&failCount, 1)
        fmt.Println("发送请求失败:", err)
        return
    }
    defer resp.Body.Close()

    // 读取响应内容（这里不实际读取内容，仅检查状态码）
    if resp.StatusCode >= 200 && resp.StatusCode < 300 {
        atomic.AddUint64(&successCount, 1)
    } else {
        atomic.AddUint64(&failCount, 1)
    }
}
