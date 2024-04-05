package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/tasks/v1"
)

func getConfigDir() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		log.Fatalf("Could not get current user: %v", err)
	}
	return filepath.Join(configDir, "gtasks")
}

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tokFile := filepath.Join(getConfigDir(), "token.json")
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func main() {
	ctx := context.Background()
	b, err := os.ReadFile(filepath.Join(getConfigDir(), "credentials.json"))
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, tasks.TasksScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(config)

	srv, err := tasks.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to retrieve tasks client: %v", err)
	}
	
	tasklistIds := make(map[string]string)
	tasklists, err := srv.Tasklists.List().Do()
	if err != nil {
		log.Fatalf("Unable to retrieve tasks lists: %v", err)
	}
	for _, item := range tasklists.Items {
		tasklistIds[item.Title] = item.Id
	}

	flag.Parse()
	cmd := flag.Arg(0)
	tasklistName := flag.Arg(1)
	tasklistId := tasklistIds[tasklistName]
	if tasklistId == "" {
		log.Fatalf("Tasklist does not exist: %s", tasklistName)
	}

	switch cmd {
	case "add":
		title := flag.Arg(2)
		notes := ""
		due := ""
		if flag.NArg() > 3 {
			notes = flag.Arg(3)
		}
		if flag.NArg() > 4 {
			due = flag.Arg(4)
		}
		_, err := srv.Tasks.Insert(tasklistId, &tasks.Task{
			Title: title,
			Notes: notes,
			Due: due,
		}).Do()
		if err != nil {
			log.Fatalf("Could not add task: %v", err)
		}
	case "list":
		tasks, err := srv.Tasks.List(tasklistId).ShowHidden(true).Do()
		if err != nil {
			log.Fatalf("Could not list tasklist items: %v", err)
		}
		bs, err := json.Marshal(tasks.Items)
		if err != nil {
			log.Fatalf("Failure when marshaling items: %v", err)
		}
		fmt.Print(string(bs))
	case "check":
		taskId := flag.Arg(2)
		task, err := srv.Tasks.Get(tasklistId, taskId).Do()
		if err != nil {
			log.Fatalf("Retrieving task failed: %v", err)
		}
		task.Status = "completed"
		_, err = srv.Tasks.Update(tasklistId, taskId, task).Do()
		if err != nil {
			log.Fatalf("Update task failed: %v", err)
		}
	case "uncheck":
		taskId := flag.Arg(2)
		task, err := srv.Tasks.Get(tasklistId, taskId).Do()
		if err != nil {
			log.Fatalf("Retrieving task failed: %v", err)
		}
		task.Status = "needsAction"
		_, err = srv.Tasks.Update(tasklistId, taskId, task).Do()
		if err != nil {
			log.Fatalf("Update task failed: %v", err)
		}
	case "delete":
		taskId := flag.Arg(2)
		if err := srv.Tasks.Delete(tasklistId, taskId).Do(); err != nil {
			log.Fatalf("Clear Delete: %v", err)
		}
	}
}
