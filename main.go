package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

type Job struct {
	ID       string
	CreateAt time.Time
	UpdateAt time.Time
	Status   string
}

type JobsApi struct {
	JobManager
}

func (ja JobsApi) HandleCreateJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}
	job, err := ja.JobManager.CreateJob()
	if err != nil {
		http.Error(w, "Failed to create job", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(job)
}

func (ja JobsApi) HandleStatusJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}
	type QueueJobRequest struct {
		JobID string `json:"job_id"`
	}
	var req QueueJobRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	job, err := ja.JobManager.StatusJob(req.JobID)
	if err != nil {
		http.Error(w, "Failed to queue job", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(job)
}

type JobManager struct {
	JobQueue chan<- Job
	DBCon    *sql.DB
}

func (jm JobManager) CreateJob() (Job, error) {
	insertUserSQL := `INSERT INTO jobs (id, createAt, updateAt, status) VALUES (?, ?, ?, ?)`
	currentTime := time.Now().UTC()
	newJob := Job{
		ID:       uuid.New().String(),
		CreateAt: currentTime,
		UpdateAt: currentTime,
		Status:   "pending",
	}
	_, err := jm.DBCon.Exec(insertUserSQL, newJob.ID, newJob.CreateAt, newJob.UpdateAt, newJob.Status)
	if err != nil {
		return newJob, err
	}
	jm.JobQueue <- newJob
	return newJob, nil
}

func (jm JobManager) StatusJob(jobID string) (Job, error) {
	sqlStatement := `SELECT * FROM jobs WHERE id = ?`
	row := jm.DBCon.QueryRow(sqlStatement, jobID)
	var resultJob Job
	err := row.Scan(&resultJob.ID, &resultJob.CreateAt, &resultJob.UpdateAt, &resultJob.Status)
	if err != nil {
		if err == sql.ErrNoRows {
			return resultJob, fmt.Errorf("job with id %s not found", jobID)
		}
		return resultJob, err
	}
	return resultJob, nil
}

type JobExecutor struct {
	JobQueue <-chan Job
	DBCon    *sql.DB
}

func (je JobExecutor) Execute() {
	for job := range je.JobQueue {
		go func(j Job) {
			previousState := j.Status
			time.Sleep(5 * time.Second)
			sqlStatement := `UPDATE jobs SET status = ? WHERE id = ?`
			result, err := je.DBCon.Exec(sqlStatement, "completed", j.ID)
			if err != nil {
				log.Println(fmt.Errorf("failed to execute update: %w", err))
				return
			}
			rowsAffected, err := result.RowsAffected()
			if err != nil {
				log.Println(fmt.Errorf("failed to retrieve rows affected: %w", err))
				return
			}
			if rowsAffected == 0 {
				log.Println(fmt.Errorf("no job found with id %s", j.ID))
			}
			selectSQL := `SELECT id, status, createAt, updateAt FROM jobs WHERE id = ?`
			var updatedJob Job
			err = je.DBCon.QueryRow(selectSQL, j.ID).Scan(&updatedJob.ID, &updatedJob.Status, &updatedJob.CreateAt, &updatedJob.UpdateAt)
			if err != nil {
				log.Println(fmt.Errorf("failed to retrieve updated job: %w", err))
			}
			log.Printf("updated job %s from %s to %s\n", j.ID, previousState, updatedJob.Status)
		}(job)
	}
}

func main() {
	db, err := sql.Open("sqlite3", "./test.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	jobQueue := make(chan Job, 5)
	jm := JobManager{
		JobQueue: jobQueue,
		DBCon:    db,
	}
	jobExecutor := JobExecutor{
		DBCon:    db,
		JobQueue: jobQueue,
	}
	go jobExecutor.Execute()
	jobsApi := JobsApi{JobManager: jm}
	http.HandleFunc("/create-job", jobsApi.HandleCreateJob)
	http.HandleFunc("/status-job", jobsApi.HandleStatusJob)

	log.Println("Server is listening on port 8080...")
	http.ListenAndServe(":8080", nil)
}
