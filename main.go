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
	Path string
	Ext  string
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

		toUpdate = append(toUpdate, FileUpdate{
			Path: path,
			Ext:  ext,
		})

		return nil
	})

	// 2. Show summary
	if len(toUpdate) == 0 {
		fmt.Println(colorYellow + "No MP3 files found." + colorReset)
		return
	}

	fmt.Println(colorCyan + "The following MP3 files will have their metadata removed:\n" + colorReset)

	for _, item := range toUpdate {
		fmt.Printf(" - %s\n", item.Path)
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

		removeMetadata(item.Path)
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

func removeMetadata(path string) {
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

	// Remove attached pictures (cover art)
	tagFile.DeleteFrames("APIC")
	tagFile.DeleteFrames("PIC")

	// Remove copyright frame
	tagFile.DeleteFrames("TCOP")

	// Remove common URL/where-from frames
	tagFile.DeleteFrames("WXXX") // user defined URL
	tagFile.DeleteFrames("WOAF") // official audio file webpage
	tagFile.DeleteFrames("WOAR") // official artist webpage
	tagFile.DeleteFrames("WOAS") // official audio source webpage
	tagFile.DeleteFrames("WORS") // official internet radio station homepage
	tagFile.DeleteFrames("WCOM") // commercial information
	tagFile.DeleteFrames("WPUB") // publisher webpage

	if err := tagFile.Save(); err != nil {
		fmt.Printf(colorRed+"Error saving MP3: %s\n"+colorReset, path)
	}
}
