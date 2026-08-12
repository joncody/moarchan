package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"html"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"moarchan/frame"
)

var app *frame.App
var tagRegex = regexp.MustCompile(`&gt;&gt;([A-Za-z0-9]+)`)

type Thread struct {
	Hash           string           `json:"hash"`
	Topic          string           `json:"topic"`
	Name           string           `json:"name"`
	Subject        string           `json:"subject"`
	Options        string           `json:"options"`
	Comment        string           `json:"comment"`
	FileName       string           `json:"file_name"`
	FileMime       string           `json:"file_mime"`
	FileSize       string           `json:"file_size"`
	FileDimensions string           `json:"file_dimensions"`
	Timestamp      string           `json:"timestamp"`
	Replies        map[string]Reply `json:"replies"`
	TaggedBy       []string         `json:"taggedBy"`
	Tagging        []string         `json:"tagging"`
}

type Reply struct {
	Hash           string   `json:"hash"`
	Thread         string   `json:"thread"`
	Topic          string   `json:"topic"`
	Name           string   `json:"name"`
	Options        string   `json:"options"`
	Comment        string   `json:"comment"`
	FileName       string   `json:"file_name"`
	FileMime       string   `json:"file_mime"`
	FileSize       string   `json:"file_size"`
	FileDimensions string   `json:"file_dimensions"`
	Timestamp      string   `json:"timestamp"`
	TaggedBy       []string `json:"taggedBy"`
	Tagging        []string `json:"tagging"`
}

type FileDetails struct {
	Name       string
	Path       string
	Mime       string
	Size       string
	Dimensions string
}

func sanitizeComment(raw string) (string, []string) {
	escaped := html.EscapeString(raw)
	escaped = strings.ReplaceAll(escaped, "\r\n", "\n")

	var tags []string
	lines := strings.Split(escaped, "\n")
	formattedLines := make([]string, 0, len(lines))

	for _, line := range lines {
		formattedLine := tagRegex.ReplaceAllStringFunc(line, func(match string) string {
			postHash := match[8:] // strip &gt;&gt;
			tags = append(tags, postHash)
			return fmt.Sprintf(`<span class="post-tag blue-text-link" data-tag="%s">%s</span>`, postHash, match)
		})

		if strings.HasPrefix(formattedLine, "&gt;") && !strings.HasPrefix(formattedLine, "&gt;&gt;") {
			formattedLine = fmt.Sprintf(`<span class="post-quote">%s</span>`, formattedLine)
		}

		formattedLines = append(formattedLines, formattedLine)
	}

	return strings.Join(formattedLines, "<br />"), tags
}

func processMultipartUpload(fileHeader *multipart.FileHeader, uniqueID string) (*FileDetails, error) {
	if fileHeader == nil {
		return &FileDetails{}, nil
	}

	file, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("open uploaded file: %w", err)
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	allowedExts := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".gif":  true,
	}
	if !allowedExts[ext] {
		return nil, fmt.Errorf("file extension %q is not allowed", ext)
	}

	config, format, err := image.DecodeConfig(file)
	if err != nil {
		return nil, fmt.Errorf("invalid or corrupted image binary: %w", err)
	}

	if _, err := file.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("reset file pointer: %w", err)
	}

	baseName := filepath.Base(fileHeader.Filename)
	if baseName == "" || baseName == "." {
		baseName = "upload" + ext
	}
	safeName := fmt.Sprintf("%s_%s", uniqueID, baseName)
	savePath := fmt.Sprintf("./static/images/uploads/%s", safeName)

	if err := os.MkdirAll("./static/images/uploads", 0755); err != nil {
		return nil, fmt.Errorf("create uploads directory: %w", err)
	}

	out, err := os.OpenFile(savePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return nil, fmt.Errorf("create destination file: %w", err)
	}
	defer out.Close()

	written, err := io.Copy(out, file)
	if err != nil {
		return nil, fmt.Errorf("write uploaded file to disk: %w", err)
	}

	return &FileDetails{
		Name:       safeName,
		Path:       savePath,
		Mime:       "image/" + format,
		Size:       fmt.Sprintf("%.1f", float64(written)/1024.0),
		Dimensions: fmt.Sprintf("%dx%d", config.Width, config.Height),
	}, nil
}

func generateUnique() (string, string) {
	now := time.Now()
	timestamp := fmt.Sprintf(
		"%d/%d/%d(%s)%02d:%02d:%02d",
		now.Month(),
		now.Day(),
		now.Year(),
		now.Weekday().String()[:3],
		now.Hour(),
		now.Minute(),
		now.Second(),
	)
	id, err := uuid.NewRandom()
	if err != nil {
		return timestamp, fmt.Sprintf("%x", sha256.Sum256([]byte(timestamp)))[:9]
	}
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(timestamp+id.String())))[:9]
	return timestamp, hash
}

func handleCreateThread(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "Multipart form parse error or size limit exceeded", http.StatusBadRequest)
		return
	}

	topic := strings.TrimSpace(r.FormValue("topic"))
	rawName := strings.TrimSpace(r.FormValue("name"))
	rawSub := strings.TrimSpace(r.FormValue("subject"))
	rawOpt := strings.TrimSpace(r.FormValue("options"))
	rawCom := strings.TrimSpace(r.FormValue("comment"))

	if topic == "" {
		http.Error(w, "Missing topic", http.StatusBadRequest)
		return
	}

	_, fileHeader, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Image file required to post a new thread", http.StatusBadRequest)
		return
	}

	timestamp, hash := generateUnique()
	fileDetails, err := processMultipartUpload(fileHeader, hash)
	if err != nil {
		log.Println("File upload error:", err)
		http.Error(w, fmt.Sprintf("File upload failed: %v", err), http.StatusBadRequest)
		return
	}

	comment, _ := sanitizeComment(rawCom)
	name := html.EscapeString(rawName)
	if name == "" {
		name = "Anonymous"
	}
	subject := html.EscapeString(rawSub)

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	const insertQuery = `
		INSERT INTO threads (hash, topic, name, subject, options, comment, file_name, file_mime, file_size, file_dimensions, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err = app.Driver.ExecContext(ctx, insertQuery,
		hash, topic, name, subject, rawOpt, comment,
		fileDetails.Name, fileDetails.Mime, fileDetails.Size, fileDetails.Dimensions, timestamp,
	)
	if err != nil {
		log.Println("Insert thread error:", err)
		http.Error(w, "Database error creating thread", http.StatusInternalServerError)
		return
	}

	thread := Thread{
		Hash:           hash,
		Topic:          topic,
		Name:           name,
		Subject:        subject,
		Options:        rawOpt,
		Comment:        comment,
		FileName:       fileDetails.Name,
		FileMime:       fileDetails.Mime,
		FileSize:       fileDetails.Size,
		FileDimensions: fileDetails.Dimensions,
		Timestamp:      timestamp,
		Replies:        make(map[string]Reply),
		TaggedBy:       []string{},
		Tagging:        []string{},
	}

	payload, err := json.Marshal(&thread)
	if err != nil {
		http.Error(w, "Serialization error", http.StatusInternalServerError)
		return
	}

	app.Hub.Broadcast(topic, "new-thread", payload)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(payload)
}

func handleCreateReply(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		http.Error(w, "Multipart form parse error or size limit exceeded", http.StatusBadRequest)
		return
	}

	topic := strings.TrimSpace(r.FormValue("topic"))
	threadHash := strings.TrimSpace(r.FormValue("thread"))
	rawName := strings.TrimSpace(r.FormValue("name"))
	rawOpt := strings.TrimSpace(r.FormValue("options"))
	rawCom := strings.TrimSpace(r.FormValue("comment"))

	if topic == "" || threadHash == "" {
		http.Error(w, "Missing thread or topic parameter", http.StatusBadRequest)
		return
	}
	if rawCom == "" {
		http.Error(w, "Comment is required to post a reply", http.StatusBadRequest)
		return
	}

	var fileHeader *multipart.FileHeader
	file, header, err := r.FormFile("file")
	if err == nil {
		file.Close()
		fileHeader = header
	}

	timestamp, hash := generateUnique()
	fileDetails, err := processMultipartUpload(fileHeader, hash)
	if err != nil {
		log.Println("File upload error:", err)
		http.Error(w, fmt.Sprintf("File upload failed: %v", err), http.StatusBadRequest)
		return
	}

	comment, tags := sanitizeComment(rawCom)
	name := html.EscapeString(rawName)
	if name == "" {
		name = "Anonymous"
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	tx, err := app.Driver.BeginTx(ctx, nil)
	if err != nil {
		http.Error(w, "Transaction error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	var exists bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM threads WHERE hash = $1 FOR UPDATE)`, threadHash).Scan(&exists); err != nil || !exists {
		http.Error(w, "Target thread not found", http.StatusNotFound)
		return
	}

	taggingJSON, _ := json.Marshal(tags)
	taggedByJSON, _ := json.Marshal([]string{})

	const insertPostQuery = `
		INSERT INTO posts (hash, thread_hash, topic, name, options, comment, file_name, file_mime, file_size, file_dimensions, timestamp, tagging, tagged_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`
	_, err = tx.ExecContext(ctx, insertPostQuery,
		hash, threadHash, topic, name, rawOpt, comment,
		fileDetails.Name, fileDetails.Mime, fileDetails.Size, fileDetails.Dimensions, timestamp,
		taggingJSON, taggedByJSON,
	)
	if err != nil {
		log.Println("Insert post error:", err)
		http.Error(w, "Database error creating reply", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "Commit transaction error", http.StatusInternalServerError)
		return
	}

	reply := Reply{
		Hash:           hash,
		Thread:         threadHash,
		Topic:          topic,
		Name:           name,
		Options:        rawOpt,
		Comment:        comment,
		FileName:       fileDetails.Name,
		FileMime:       fileDetails.Mime,
		FileSize:       fileDetails.Size,
		FileDimensions: fileDetails.Dimensions,
		Timestamp:      timestamp,
		TaggedBy:       []string{},
		Tagging:        tags,
	}

	payload, err := json.Marshal(&reply)
	if err != nil {
		http.Error(w, "Serialization error", http.StatusInternalServerError)
		return
	}

	app.Hub.Broadcast(topic, "new-reply", payload)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(payload)
}

func main() {
	var err error
	app, err = frame.NewApp()
	if err != nil {
		log.Fatal(err)
	}

	// 1. Programmatic Route Declarations (Fluent Go API)
	app.Route("/news").Template("news")
	app.Route("/blog").Template("blog")
	app.Route("/faq").Template("faq")
	app.Route("/rules").Template("rules")
	app.Route("/advertise").Template("advertise")
	app.Route("/press").Template("press")
	app.Route("/about").Template("about")
	app.Route("/feedback").Template("feedback")
	app.Route("/legal").Template("legal")
	app.Route("/contact").Template("contact")

	app.Route(`^/([A-Za-z0-9]+)/thread/([A-Za-z0-9]+)$`).
		Table("$1").
		Key("$2").
		Template("thread").
		Controller("service")

	app.Route(`^/([A-Za-z0-9]+)[\/]?$`).
		Table("$1").
		Template("topic").
		Controller("service")

	app.Route(`^/$`).
		Table("main").
		Template("main").
		Controller("main")

	// 2. Application Data Resolver Hook
	app.DataProvider = MoarchanDataProvider

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 3. Database Schema Initialization
	if err := InitMoarchanSchema(ctx, app); err != nil {
		log.Fatalf("Failed to initialize moarchan schema: %v", err)
	}

	// 4. REST Endpoints
	app.Router.HandleFunc("/api/threads", handleCreateThread).Methods("POST")
	app.Router.HandleFunc("/api/replies", handleCreateReply).Methods("POST")

	if err := app.Start(); err != nil {
		log.Fatal("Application start error:", err)
	}
}
