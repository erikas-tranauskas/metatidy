package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bogem/id3v2"
	"github.com/dhowden/tag"
)

var (
	dryRun = flag.Bool("dry-run", false, "Show what would be changed, but do not write anything")

	musicExts = map[string]bool{
		".mp3": true,
	}
)

const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorRed    = "\033[31m"
	colorCyan   = "\033[36m"
)

type FileUpdate struct {
	Path    string
	Comment string
	Ext     string
}

func main() {
	flag.Parse()

	exPath, err := os.Executable()
	if err != nil {
		panic(err)
	}

	root := filepath.Dir(exPath)

	var toUpdate []FileUpdate

	// 1. Scan files
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(info.Name()))
		if !musicExts[ext] {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()

		metaData, err := tag.ReadFrom(f)
		if err != nil {
			return nil
		}

		comment := metaData.Comment()
		if comment != "" {
			toUpdate = append(toUpdate, FileUpdate{
				Path:    path,
				Comment: comment,
				Ext:     ext,
			})
		}

		return nil
	})

	// 2. Show summary
	if len(toUpdate) == 0 {
		fmt.Println(colorYellow + "No MP3 files with comments found." + colorReset)
		return
	}

	fmt.Println(colorCyan + "The following MP3 files will have their COMMENT removed:\n" + colorReset)

	for _, item := range toUpdate {
		fmt.Printf(" - %s %s(comment: %q)%s\n",
			item.Path, colorYellow, item.Comment, colorReset)
	}

	fmt.Printf("\n%sTotal: %d files%s\n\n", colorGreen, len(toUpdate), colorReset)

	if *dryRun {
		fmt.Println(colorYellow + "[DRY RUN] No changes will be written.\n" + colorReset)
	}

	// 3. Confirm
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Continue? (y/N): ")

	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))

	if answer != "y" && answer != "yes" {
		fmt.Println(colorRed + "Cancelled." + colorReset)
		return
	}

	// 4. Process + progress bar
	total := len(toUpdate)
	fmt.Println()

	for i, item := range toUpdate {
		drawProgressBar(i+1, total)

		if *dryRun {
			time.Sleep(50 * time.Millisecond)
			continue
		}

		removeCommentMP3(item.Path)
	}

	fmt.Println("\n" + colorGreen + "Completed successfully." + colorReset)
}

func drawProgressBar(current, total int) {
	percent := float64(current) / float64(total)
	width := 40
	filled := int(percent * float64(width))

	bar := "[" + strings.Repeat("█", filled) + strings.Repeat(" ", width-filled) + "]"
	fmt.Printf("\r%s %d/%d", bar, current, total)
}

// ============================================================================
// MP3 (ID3v2)
// ============================================================================

func removeCommentMP3(path string) {
	tagFile, err := id3v2.Open(path, id3v2.Options{Parse: true})
	if err != nil {
		fmt.Printf(colorRed+"Error opening MP3: %s\n"+colorReset, path)
		return
	}
	defer tagFile.Close()

	// Remove comment frames
	tagFile.DeleteFrames("COMM")

	// Remove other text frames commonly used to store comments
	tagFile.DeleteFrames("TXXX")
	tagFile.DeleteFrames("USLT")
	tagFile.DeleteFrames("SYLT")

	if err := tagFile.Save(); err != nil {
		fmt.Printf(colorRed+"Error saving MP3: %s\n"+colorReset, path)
	}
}
