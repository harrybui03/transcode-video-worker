package service

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Resolution struct {
	Width     int
	Height    int
	Bitrate   string // e.g., "800k"
	AudioRate string // e.g., "96k"
}

// Define the target resolutions for HLS.
var resolutions = []Resolution{
	{Width: 256, Height: 144, Bitrate: "200k", AudioRate: "64k"},
	{Width: 640, Height: 360, Bitrate: "800k", AudioRate: "96k"},
	{Width: 854, Height: 480, Bitrate: "1500k", AudioRate: "128k"},
	{Width: 1280, Height: 720, Bitrate: "3000k", AudioRate: "192k"},
	{Width: 1920, Height: 1080, Bitrate: "5000k", AudioRate: "192k"},
}

func transcodeToHLS(inputFilepath, outputDir string) error {
	// 1. Build the filter_complex string for scaling videos
	// We use more descriptive labels like [v144], [v360] etc.
	var filterComplexBuilder strings.Builder
	for _, r := range resolutions {
		// [0:v] is the first video stream from the input file
		filterComplexBuilder.WriteString(
			fmt.Sprintf("[0:v]scale=w=%d:h=%d:force_original_aspect_ratio=decrease,pad=w=%d:h=%d:x=(ow-iw)/2:y=(oh-ih)/2[v%d]; ",
				r.Width, r.Height, r.Width, r.Height, r.Height))
	}

	// Base arguments, starting with the input file and the complex filter graph
	ffmpegArgs := []string{
		"-i", inputFilepath,
		"-filter_complex", strings.TrimSuffix(filterComplexBuilder.String(), "; "),
		"-map", "0:a:0?", // Map the first audio stream from input. The '?' makes it optional.
	}

	// 2. Add output options for EACH resolution
	for _, r := range resolutions {
		playlistName := fmt.Sprintf("%dp.m3u8", r.Height)
		segmentName := fmt.Sprintf("%dp_%%03d.ts", r.Height)

		ffmpegArgs = append(ffmpegArgs,
			// Map the corresponding scaled video stream from the filter_complex
			"-map", fmt.Sprintf("[v%d]", r.Height),

			// Set video codec, bitrate, and other options for this variant
			"-c:v", "libx264",
			"-preset", "veryfast",
			"-crf", "22", // Constant Rate Factor for quality
			"-b:v", r.Bitrate,
			"-maxrate", r.Bitrate,
			"-bufsize", r.Bitrate,

			// Set audio codec and bitrate for this variant
			"-c:a", "aac",
			"-b:a", r.AudioRate,

			// Set HLS options for this variant
			"-f", "hls",
			"-hls_time", "6", // Shorter segment time
			"-hls_playlist_type", "vod",
			"-hls_segment_filename", filepath.Join(outputDir, segmentName),
			filepath.Join(outputDir, playlistName), // Output M3U8 file for this variant
		)
	}

	cmd := exec.Command("ffmpeg", ffmpegArgs...)
	log.Printf("Executing FFmpeg command: ffmpeg %s", strings.Join(ffmpegArgs, " "))

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("FFmpeg output:\n%s\n", string(output))
		return fmt.Errorf("ffmpeg execution failed: %w", err)
	}

	return nil
}

// createMasterPlaylist creates the master M3U8 file.
// Note: Modern ffmpeg can often create the master playlist directly using `-master_pl_name`.
// This function serves as a reliable fallback or for customization.
func createMasterPlaylist(outputDir string) error {
	masterPlaylistPath := filepath.Join(outputDir, "master.m3u8")
	content := "#EXTM3U\n#EXT-X-VERSION:3\n"

	log.Println("Creating master playlist...")

	for _, r := range resolutions {
		// Calculate bandwidth (bitrate in bits per second)
		// Assuming video bitrate is in 'k', so multiply by 1000
		var videoBitrateBPS int
		fmt.Sscanf(r.Bitrate, "%dk", &videoBitrateBPS)

		// Assuming audio bitrate is in 'k', so multiply by 1000
		var audioBitrateBPS int
		fmt.Sscanf(r.AudioRate, "%dk", &audioBitrateBPS)

		// Total bandwidth for the variant
		totalBandwidth := (videoBitrateBPS + audioBitrateBPS) * 1000

		playlistName := fmt.Sprintf("%dp.m3u8", r.Height)
		content += fmt.Sprintf("#EXT-X-STREAM-INF:BANDWIDTH=%d,RESOLUTION=%dx%d\n", totalBandwidth, r.Width, r.Height)
		content += playlistName + "\n"
	}

	return os.WriteFile(masterPlaylistPath, []byte(content), 0644)
}
