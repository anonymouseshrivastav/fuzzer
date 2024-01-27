package main

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	mutex               sync.Mutex
	wg                  sync.WaitGroup
	status_codes_string     = "404"
	errors              int = 0
	checkedLines        int = 0
)

func main() {
	banner()
	if len(os.Args) < 4 {
		usage("Provide all values")
	}

	var (
		URL          = os.Args[1]
		fileLocation = os.Args[2]
		threadsSTR   = os.Args[3]
	)

	if len(os.Args) == 5 {
		status_codes_string = os.Args[4]
	}

	status_codes := strings.Split(status_codes_string, ",")
	threads, err := strconv.Atoi(threadsSTR)

	if err != nil {
		usage("Invalid arguments")
	}

	semaphore := make(chan struct{}, threads)

	file, err := os.Open(fileLocation)
	if err != nil {
		usage("Invalid file location")
	}
	defer file.Close()

	totalLines := checkTotalLines(fileLocation)
	scanner := bufio.NewScanner(file)
	fmt.Println("\033[36m", "Code:Length\tProcess\tErrors\tURL", "\033[0m")
	for scanner.Scan() {
		word := scanner.Text()
		wg.Add(1)
		go start(URL, word, semaphore, status_codes, totalLines)
	}

	wg.Wait()

	fmt.Println("Fuzzing Completed..")
}

func banner() {
	fmt.Println("\033[31m", `
███████╗██╗   ██╗███████╗███████╗███████╗██████╗
██╔════╝██║   ██║╚══███╔╝╚══███╔╝██╔════╝██╔══██╗
█████╗  ██║   ██║  ███╔╝   ███╔╝ █████╗  ██████╔╝
██╔══╝  ██║   ██║ ███╔╝   ███╔╝  ██╔══╝  ██╔══██╗
██║     ╚██████╔╝███████╗███████╗███████╗██║  ██║
╚═╝      ╚═════╝ ╚══════╝╚══════╝╚══════╝╚═╝  ╚═╝`)
	fmt.Println("\tby Anon Shrivastav\n", "\033[0m")
}

func usage(msg string) {
	fmt.Println("\nError:\033[31m ", msg, "\033[0m")
	fmt.Println("\033[32mfuzz <url> <wordlist> <threads> <negative status codes>")
	fmt.Println("fuzz https://google.com wordlist.txt 40 404,500,302\033[0m")
	os.Exit(0)
}

func start(url string, word string, semaphore chan struct{}, status_codes []string, totalLines int) {
	defer wg.Done()

	semaphore <- struct{}{}
	defer func() { <-semaphore }()

	mutex.Lock()
	checkedLines++
	mutex.Unlock()

	if !strings.HasSuffix(url, "/") {
		url = url + "/"
	}

	word = strings.TrimPrefix(word, "/")
	address := url + word

	address = strings.TrimSpace(address)

	req, err := http.NewRequest("GET", address, nil)

	if err != nil {
		mutex.Lock()
		errors++
		mutex.Unlock()
		return
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/96.0.4664.45 Safari/537.36")

	client := &http.Client{Timeout: 5 * time.Second}
	res, err := client.Do(req)

	if err != nil {
		mutex.Lock()
		errors++
		mutex.Unlock()
		return
	}
	defer res.Body.Close()

	result := true
	for _, status_string := range status_codes {
		status, _ := strconv.Atoi(status_string)
		if res.StatusCode == status {
			result = false
			break
		}
	}

	checkedPercentage := (checkedLines * 100) / totalLines
	if result {
		clr := "\033[33m"
		if res.StatusCode == 200 {
			clr = "\033[32m"
		}

		fmt.Printf("%s%d:%d\t%d/%d (%d%%)\t%d\t%s\n\033[0m", clr, res.StatusCode, res.ContentLength, checkedLines, totalLines, checkedPercentage, errors, address)
	}

}

func checkTotalLines(fileName string) int {
	file, _ := os.Open(fileName)

	scanner := bufio.NewScanner(file)

	count := 0
	for scanner.Scan() {
		count++
	}

	return count
}
