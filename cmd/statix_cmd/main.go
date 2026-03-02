package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	URL   string
	Token string
}

func configPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".statix_config")
}

func saveConfig(urlStr, token string) error {
	content := fmt.Sprintf("%s\n%s\n", urlStr, token)
	return os.WriteFile(configPath(), []byte(content), 0600)
}

func loadConfig() (*Config, error) {
	b, err := os.ReadFile(configPath())
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	if len(lines) < 2 {
		return nil, fmt.Errorf("invalid config file")
	}
	return &Config{
		URL:   lines[0],
		Token: lines[1],
	}, nil
}

func publish(title string, 
             subjectID string, 
             isPublic string, 
             filePath string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	data := url.Values{}
	data.Set("title", title)
	data.Set("subject_id", subjectID)
	data.Set("is_public", isPublic)
	data.Set("html", string(content))

	req, err := http.NewRequest("POST", cfg.URL+"/admin/new", strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Statix-Token", cfg.Token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	fmt.Printf("Status: %s\n", resp.Status)
	fmt.Println(string(body))

	return nil
}

func editArticle(id, title, subjectID, isPublic, filePath string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	data := url.Values{}
	data.Set("title", title)
	data.Set("subject_id", subjectID)
	data.Set("is_public", isPublic)
	data.Set("html", string(content))

	endpoint := fmt.Sprintf("%s/admin/articles/%s", cfg.URL, id)

	req, err := http.NewRequest("POST", endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Statix-Token", cfg.Token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	fmt.Printf("Status: %s\n", resp.Status)
	fmt.Println(string(body))

	return nil
}

func listArticles() error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	req, err := http.NewRequest("GET", cfg.URL+"/admin/api/articles", nil)
	if err != nil {
		return err
	}

	req.Header.Set("X-Statix-Token", cfg.Token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned %s:\n%s", resp.Status, string(body))
	}

	fmt.Print(string(body))
	return nil
}

func main() {
	setCreds := flag.Bool("set-credentials", false, "Set credentials")
	publishFlag := flag.Bool("publish", false, "Publish article")

    articlesFlag := flag.Bool("articles", false, "List articles (id + title)")
    
    editFlag := flag.Bool("edit", false, "Edit article")
    id := flag.String("id", "", "Article ID")

	password := flag.String("password", "", "Token")
	urlStr := flag.String("url", "", "Base URL")

	title := flag.String("title", "", "Article title")
	subjectID := flag.String("subject_id", "", "Subject ID")
	isPublic := flag.String("is_public", "true", "Visibility")
	file := flag.String("file", "", "HTML file path")

	flag.Parse()

	if *setCreds {
		if *password == "" || *urlStr == "" {
			fmt.Println("Missing --password or --url")
			return
		}
		err := saveConfig(*urlStr, *password)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		fmt.Println("Credentials saved.")
		return
	}

	if *publishFlag {
		if *title == "" || *subjectID == "" || *file == "" {
			fmt.Println("Missing required publish flags.")
			return
		}
		err := publish(*title, *subjectID, *isPublic, *file)
		if err != nil {
			fmt.Println("Error:", err)
		}
		return
	}

    if *editFlag {
    	if *id == "" || *title == "" || *subjectID == "" || *file == "" {
    		fmt.Println("Missing required edit flags.")
    		return
    	}
    
    	err := editArticle(*id, *title, *subjectID, *isPublic, *file)
    	if err != nil {
    		fmt.Println("Error:", err)
    	}
    	return
    }

    if *articlesFlag {
    	if err := listArticles(); err != nil {
    		fmt.Println("Error:", err)
    	}
    	return
    }

	fmt.Println("No valid command.")
}





