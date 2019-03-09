package main

import (
	"bytes"
	"crypto"
	"crypto/hmac"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

var serverWorker *TrackerWorker

var serverClient = http.Client{
	Timeout:   config.ServerTimeout,
	Transport: new(ServerTripper),
}

var serverUserAgent = "od-database-crawler/" + rootCmd.Version

//TODO: Move those elsewhere?
type WorkerAccessRequest struct {
	Assign  bool `json:"assign"`
	Submit  bool `json:"submit"`
	Project int  `json:"project"`
}

//todo: only keep necessary info
type FetchTaskResponse struct {
	Ok      bool   `json:"ok"`
	Message string `json:"message"`
	Content struct {
		Task struct {
			Id       int64 `json:"id"`
			Priority int64 `json:"priority"`
			Project  struct {
				Id         int64      `json:"id"`
				Priority   int64      `json:"priority"`
				Name       string     `json:"name"`
				CloneUrl   string     `json:"clone_url"`
				GitRepo    string     `json:"git_repo"`
				Version    string     `json:"version"`
				Motd       string     `json:"motd"`
				Public     bool       `json:"public"`
				Hidden     bool       `json:"hidden"`
				Chain      int64      `json:"chain"`
				Paused     bool       `json:"paused"`
				AssignRate rate.Limit `json:"assign_rate"`
				SubmitRate rate.Limit `json:"submit_rate"`
			} `json:"project"`
			Assignee          int64  `json:"assignee"`
			Retries           int64  `json:"retries"`
			MaxRetries        int64  `json:"max_retries"`
			Status            int64  `json:"status"`
			Recipe            string `json:"recipe"`
			MaxAssignTime     int64  `json:"max_assign_time"`
			AssignTime        int64  `json:"assign_time"`
			VerificationCount int64  `json:"verification_count"`
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

func getOrCreateWorker() *TrackerWorker {

	var worker TrackerWorker

	if _, err := os.Stat("worker.json"); os.IsNotExist(err) {
		req := CreateTrackerWorkerRequest{
			Alias: "crawler", //todo: load from config
		}
		body, _ := json.Marshal(&req)
		buf := bytes.NewBuffer(body)
		resp, _ := serverClient.Post(config.TrackerUrl+"/worker/create", "application/json", buf)
		//todo: handle err

		fmt.Println(resp.StatusCode)

		workerResponse := CreateTrackerWorkerResponse{}
		respBody, _ := ioutil.ReadAll(resp.Body)
		_ = json.Unmarshal(respBody, &workerResponse)
		//todo handle err
		fmt.Println(workerResponse)

		workerJsonData, _ := json.Marshal(&workerResponse.Content.Worker)
		fp, _ := os.OpenFile("worker.json", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
		_, _ = fp.Write(workerJsonData)
		//todo: handle err
	} else {
		fp, _ := os.OpenFile("worker.json", os.O_RDONLY, 0600)
		workerJsonData, _ := ioutil.ReadAll(fp)
		_ = json.Unmarshal(workerJsonData, &worker)
		//todo: handle err
	}

	return &worker
}

func FetchTask() (t *Task, err error) {

	//todo: this whole block should definitely be extracted elsewhere
	if serverWorker == nil {
		serverWorker = getOrCreateWorker()

		//Request ASSIGN permission
		//todo: project ID should be stored as a int in the first place
		pid, _ := strconv.Atoi(config.TrackerProject)
		accessReq, _ := json.Marshal(WorkerAccessRequest{
			Project: int(pid),
			Assign:  true,
			Submit:  false,
		})
		buf := bytes.NewBuffer(accessReq)
		res, err := serverClient.Post(config.TrackerUrl+"/project/request_access", "application/json", buf)
		if err != nil {
			panic(err)
		}
		fmt.Println(res.StatusCode)
	}

	res, err := serverClient.Get(
		config.TrackerUrl + "/task/get/" + config.TrackerProject)

	if err != nil {
		return
	}
	defer res.Body.Close()

	switch res.StatusCode {
	case 200:
		break
		//TODO: 404 should not happen.
	case 404, 500:
		return nil, nil
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
		//The tracker will return Ok=false when no tasks are available
		err = errors.New(jsonResponse.Message)
		return
	}

	fmt.Println(jsonResponse.Content.Task.Recipe)
	task := Task{}
	err = json.Unmarshal([]byte(jsonResponse.Content.Task.Recipe), &task)
	if _, ok := err.(*json.SyntaxError); ok {
		return nil, fmt.Errorf("/task/get returned invalid JSON")
	} else if err != nil {
		return
	}

	t = &task

	return
}

func PushResult(result *TaskResult, f *os.File) (err error) {
	if result.WebsiteId == 0 {
		// Not a real result, don't push
		return nil
	}

	// Rewind to the beginning of the file
	_, err = f.Seek(0, 0)
	if err != nil {
		return
	}

	err = uploadWebsocket(result.WebsiteId, f)
	if err != nil {
		logrus.Errorf("Failed to upload file list: %s", err)
		err2 := CancelTask(result.WebsiteId)
		if err2 != nil {
			logrus.Error(err2)
		}
		return
	}

	// Upload result ignoring errors
	uploadResult(result)

	return
}

func uploadWebsocket(websiteId uint64, f *os.File) error {

	//TODO:
	/*
	 * OD-DB will give you an Upload token when you fetch the task
	 * Open a WS connection at {ws_bucket_addr}/upload with the 'X-Upload-Token' as header
	 * Stream whole file as a single WS message
	 * Close connection
	 */
	return nil
}

func uploadResult(result *TaskResult) (err error) {

	//TODO:
	/*
	 * When the file has been uploaded, just release the task with the TR_OK code
	 * Don't bother sending the ODDB-related stuff, You just need the task id
	 * Probably a good idea to wrap this around a new releaseTask() function
	 */

	req := releaseTaskRequest{
		TaskId:     int64(result.WebsiteId),
		ResultCode: result.ResultCode,
		// TODO What is verification
		Verification: 0,
	}

	resultEnc, err := json.Marshal(&req)
	if err != nil {
		panic(err)
	}

	res, err := serverClient.PostForm(
		config.TrackerUrl+"/task/release",
		url.Values{
			"token":  {config.Token},
			"result": {string(resultEnc)},
		},
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

func CancelTask(websiteId uint64) (err error) {

	//TODO: Maybe wrap this function around releaseTask(cancel: bool) ?
	return
}

type ServerTripper struct{}

func (t *ServerTripper) RoundTrip(req *http.Request) (res *http.Response, err error) {
	req.Header.Set("User-Agent", serverUserAgent)

	//TODO: Move this whole block elsewhere
	if serverWorker != nil {
		var content []byte
		if req.Method == "GET" {
			content = []byte("/task/get/" + config.TrackerProject) //todo: find a less retarded way of doing that
		} else {
			//todo: this is retarded and should be moved elsewhere
			buf, _ := ioutil.ReadAll(req.Body)
			rdr1 := ioutil.NopCloser(bytes.NewBuffer(buf))
			rdr2 := ioutil.NopCloser(bytes.NewBuffer(buf))

			content, _ = ioutil.ReadAll(rdr1)

			req.Body = rdr2
		}

		ts := time.Now().Format(time.RFC1123)

		mac := hmac.New(crypto.SHA256.New, serverWorker.Secret)
		mac.Write(content)
		mac.Write([]byte(ts))
		sig := hex.EncodeToString(mac.Sum(nil))

		req.Header.Add("X-Worker-Id", strconv.Itoa(serverWorker.Id))
		req.Header.Add("Timestamp", time.Now().Format(time.RFC1123))
		req.Header.Add("X-Signature", sig)

	}
	return http.DefaultTransport.RoundTrip(req)
}

const mimeJSON = "application/json"

// https://github.com/simon987/task_tracker/blob/master/api/models.go

type submitTaskRequest struct {
	Project           int64  `json:"project"`
	MaxRetries        int64  `json:"max_retries"`
	Recipe            string `json:"recipe"`
	Priority          int64  `json:"priority"`
	MaxAssignTime     int64  `json:"max_assign_time"`
	Hash64            int64  `json:"hash_u64"`
	UniqueString      string `json:"unique_string"`
	VerificationCount int64  `json:"verification_count"`
}

type releaseTaskRequest struct {
	TaskId       int64      `json:"task_id"`
	ResultCode   ResultCode `json:"result"`
	Verification int64      `json:"verification"`
}
