package main

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gocolly/colly"
	"github.com/signintech/gopdf"
)

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting executable path: %s\n", err)
		return
	}

	url := read_input()
	chapters := []string{}
	if strings.Contains(url, "mangas") {
		get_every_chapters_url(url, &chapters)
	} else {
		chapters = append(chapters, url)
	}

	for _, chapter_url := range chapters {
		c := colly.NewCollector()
		chapter_info := extractChapterInfo(chapter_url)
		chapter_dir := fmt.Sprintf("%s/%s", cwd, chapter_info)

		create_download_dir(chapter_dir)

		chapter_images := []string{}

		c.OnHTML("img", func(e *colly.HTMLElement) {
			src := e.Attr("src")
			chapter_images = append(chapter_images, src)
			// err := download_image(src, chapter_dir)
			fmt.Println(err)
		})

		c.OnRequest(func(r *colly.Request) {
			fmt.Println("Visiting", r.URL)
		})

		c.Visit(chapter_url)
		downloadImages(chapter_images, chapter_dir)
		outputFile := fmt.Sprintf("%s.pdf", chapter_info)
		jpgFiles, err := getJPGFiles(chapter_dir)
		if err != nil {
			fmt.Printf("Error retrieving JPG files: %s\n", err)
			return
		}

		err = createPDFFromImages(jpgFiles, outputFile)
		if err != nil {
			fmt.Printf("Error creating PDF: %s\n", err)
			return
		}

		fmt.Printf("PDF created: %s\n", outputFile)
	}

}

func getJPGFiles(directory string) ([]string, error) {
	var jpgFiles []string

	files, err := ioutil.ReadDir(directory)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if !file.IsDir() {
			if strings.HasSuffix(strings.ToLower(file.Name()), ".jpg") {
				jpgFiles = append(jpgFiles, filepath.Join(directory, file.Name()))
			}
		}
	}

	return jpgFiles, nil
}
func downloadImages(imageURLs []string, chapter_dir string) {
	var wg sync.WaitGroup
	ch := make(chan string)

	for _, imageURL := range imageURLs {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			err := download_image(url, chapter_dir)
			if err != nil {
				fmt.Printf("Error downloading image: %s\n", err)
			}
			ch <- url
		}(imageURL)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	for downloadedURL := range ch {
		fmt.Printf("Image downloaded: %s\n", downloadedURL)
	}
}

func create_download_dir(chapter_dir string) {
	err := os.Mkdir(chapter_dir, os.ModePerm)
	if err != nil {
		fmt.Printf("Error creating directory: %s\n", err)
	}
}

func extractChapterInfo(urlString string) string {
	parsedURL, err := url.Parse(urlString)
	if err != nil {
		fmt.Printf("Error parsing URL: %s\n", err)
		return ""
	}

	lastPart := path.Base(parsedURL.Path)
	// Remove "review" and everything after it
	lastPart = strings.TrimSuffix(lastPart, "-review")
	// Remove leading and trailing hyphens
	lastPart = strings.TrimPrefix(lastPart, "-")
	lastPart = strings.TrimSuffix(lastPart, "-")

	return lastPart
}

func get_last_part_of_url(urlString string) string {
	parsedURL, err := url.Parse(urlString)
	if err != nil {
		fmt.Printf("Error parsing URL: %s\n", err)
		return ""
	}

	lastPart := path.Base(parsedURL.Path)
	return lastPart
}
func download_image(imageURL, downloadDir string) error {
	resp, err := http.Get(imageURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	filename := get_last_part_of_url(imageURL)
	filepath := filepath.Join(downloadDir, filename)
	fmt.Printf("Downloading %s to %s\n", imageURL, filepath)

	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func createPDFFromImages(imageFiles []string, outputFile string) error {
	pdf := gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})

	for _, imageFile := range imageFiles {
		err := addImageToPDF(&pdf, imageFile)
		if err != nil {
			return err
		}
	}

	err := pdf.WritePdf(outputFile)
	if err != nil {
		return err
	}

	return nil
}

func addImageToPDF(pdf *gopdf.GoPdf, imageFile string) error {
	pdf.AddPage()
	pdf.Image(imageFile, 0, 0, nil)

	return nil
}

func get_every_chapters_url(url string, chapters *[]string) {
	c := colly.NewCollector()

	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		if strings.Contains(e.Attr("href"), "chapters") {
			*chapters = append(*chapters, fmt.Sprintf("https://tcbscans.com%s", e.Attr("href")))
		}
	})

	c.Visit(url)
}

func read_input() string {
	fmt.Print("Enter text: ")
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("An error occured while reading input. Please try again", err)
		return ""
	}

	input = strings.TrimSuffix(input, "\n")
	return input
}
