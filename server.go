package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/fasthttp/websocket"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
)

var serverWorker *TrackerWorker

var serverClient = http.Client{
	Timeout:   config.ServerTimeout,
	Transport: new(ServerTripper),
}

var serverUserAgent = "od-database-crawler/" + rootCmd.Version

func getOrCreateWorker() {

	if _, err := os.Stat("worker.json"); os.IsNotExist(err) {
		req := CreateTrackerWorkerRequest{
			Alias: config.TrackerAlias,
		}
		body, _ := json.Marshal(&req)
		buf := bytes.NewBuffer(body)
		resp, _ := serverClient.Post(config.TrackerUrl+"/worker/create", "application/json", buf)

		workerResponse := CreateTrackerWorkerResponse{}
		respBody, _ := ioutil.ReadAll(resp.Body)
		_ = json.Unmarshal(respBody, &workerResponse)

		workerJsonData, _ := json.Marshal(&workerResponse.Content.Worker)
		fp, _ := os.OpenFile("worker.json", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
		_, _ = fp.Write(workerJsonData)

		//Request ASSIGN permission
		serverWorker = &workerResponse.Content.Worker
		accessReq, _ := json.Marshal(WorkerAccessRequest{
			Project: config.TrackerProject,
			Assign:  true,
			Submit:  false,
		})
		buf = bytes.NewBuffer(accessReq)
		res, err := serverClient.Post(config.TrackerUrl+"/project/request_access", "application/json", buf)
		if err != nil {
			panic(err)
		}
		logrus.WithFields(logrus.Fields{
			"response": res.StatusCode,
		}).Info("Requested ASSIGN permission")
	} else {
		var worker TrackerWorker

		fp, _ := os.OpenFile("worker.json", os.O_RDONLY, 0600)
		workerJsonData, _ := ioutil.ReadAll(fp)
		_ = json.Unmarshal(workerJsonData, &worker)

		serverWorker = &worker
	}
}

func FetchTask() (t *Task, err error) {

	if serverWorker == nil {
		getOrCreateWorker()
	}

	res, err := serverClient.Get(config.TrackerUrl + "/task/get/" + strconv.Itoa(config.TrackerProject))

	if err != nil {
		return
	}
	defer res.Body.Close()

	switch res.StatusCode {
	case 200:
		break
	default:
		return nil, fmt.Errorf("http %s", res.Status)
	}

	jsonResponse := FetchTaskResponse{}
	err = json.NewDecoder(res.Body).Decode(&jsonResponse)
	if _, ok := err.(*json.SyntaxError); ok {
		return nil, fmt.Errorf("/task/get returned invalid JSON")
	} else if err != nil {
		return
	}

	if !jsonResponse.Ok {
		if jsonResponse.Message == "No task available" {
			return nil, nil
		}
		return nil, errors.New(jsonResponse.Message)
	}

	task := Task{}
	err = json.Unmarshal([]byte(jsonResponse.Content.Task.Recipe), &task)
	if _, ok := err.(*json.SyntaxError); ok {
		return nil, fmt.Errorf("/task/get returned invalid JSON")
	} else if err != nil {
		return
	}

	t = &task
	t.TaskId = jsonResponse.Content.Task.Id

	return
}

func PushResult(task *Task, f *os.File) (err error) {
	if task.WebsiteId == 0 {
		// Not a real result, don't push
		return nil
	}

	// Rewind to the beginning of the file
	_, err = f.Seek(0, 0)
	if err != nil {
		return
	}

	err = uploadWebsocket(f, task.UploadToken)
	if err != nil {
		logrus.Errorf("Failed to upload file list: %s", err)
		err2 := releaseTask(task, TR_FAIL)
		if err2 != nil {
			logrus.Error(err2)
		}
		return
	}

	// Upload result ignoring errors
	_ = releaseTask(task, TR_OK)

	return
}

func uploadWebsocket(f *os.File, token string) (err error) {

	u := url.URL{Scheme: config.WsBucketScheme, Host: config.WsBucketHost, Path: "/upload"}

	header := http.Header{}
	header.Add("X-Upload-Token", token)
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), header)
	if err != nil {
		return
	}

	conn.EnableWriteCompression(true) //TODO: Is this necessary?

	socketWriter, _ := conn.NextWriter(websocket.BinaryMessage)
	_, _ = io.Copy(socketWriter, f)
	err = socketWriter.Close()
	if err != nil {
		logrus.Error("FIXME: couldn't do file upload")
		return
	}
	err = conn.Close()
	if err != nil {
		return
	}

	return nil
}

func releaseTask(task *Task, taskResult ResultCode) (err error) {

	req := releaseTaskRequest{
		TaskId:     task.TaskId,
		ResultCode: taskResult,
		// TODO Will implement verification in a later ODDB update
		Verification: 0,
	}

	resultEnc, err := json.Marshal(&req)
	if err != nil {
		panic(err)
	}
	body := bytes.NewBuffer(resultEnc)

	res, err := serverClient.Post(
		config.TrackerUrl+"/task/release",
		"application/json",
		body,
	)
	if err != nil {
		return
	}
	res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return HttpError{res.StatusCode}
	}

	return
}

type ServerTripper struct{}

func (t *ServerTripper) RoundTrip(req *http.Request) (res *http.Response, err error) {
	req.Header.Set("User-Agent", serverUserAgent)

	//TODO: Use task_tracker/client ?
	if serverWorker != nil {
		req.Header.Add("X-Worker-Id", strconv.Itoa(serverWorker.Id))
		req.Header.Add("X-Secret", base64.StdEncoding.EncodeToString(serverWorker.Secret))
	}
	return http.DefaultTransport.RoundTrip(req)
}

// https://github.com/simon987/task_tracker/blob/master/api/models.go

type releaseTaskRequest struct {
	TaskId       int64      `json:"task_id"`
	ResultCode   ResultCode `json:"result"`
	Verification int64      `json:"verification"`
}

type WorkerAccessRequest struct {
	Assign  bool `json:"assign"`
	Submit  bool `json:"submit"`
	Project int  `json:"project"`
}

type FetchTaskResponse struct {
	Ok      bool   `json:"ok"`
	Message string `json:"message"`
	Content struct {
		Task struct {
			Id       int64 `json:"id"`
			Priority int64 `json:"priority"`
			Project  struct {
				Id         int64      `json:"id"`
				Name       string     `json:"name"`
				Version    string     `json:"version"`
				AssignRate rate.Limit `json:"assign_rate"`
				SubmitRate rate.Limit `json:"submit_rate"`
			} `json:"project"`
			Recipe string `json:"recipe"`
		} `json:"task"`
	} `json:"content"`
}

type TrackerWorker struct {
	Alias  string `json:"alias"`
	Id     int    `json:"id"`
	Secret []byte `json:"secret"`
}

type CreateTrackerWorkerResponse struct {
	Ok      bool   `json:"ok"`
	Message string `json:"message"`
	Content struct {
		Worker TrackerWorker `json:"worker"`
	} `json:"content"`
}

type CreateTrackerWorkerRequest struct {
	Alias string `json:"alias"`
}
