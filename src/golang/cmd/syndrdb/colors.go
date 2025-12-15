package main

import (
	"fmt"
	"os"
)

// ANSI color codes (constants)
const (
	ansiReset  = "\033[0m"
	ansiRed    = "\033[31m"
	ansiGreen  = "\033[32m"
	ansiYellow = "\033[33m"
	ansiBlue   = "\033[34m"
	ansiCyan   = "\033[36m"
	ansiBold   = "\033[1m"
	ansiDim    = "\033[2m"
)

var colorsEnabled = true

func init() {
	// Disable colors if NO_COLOR env var is set or output is not a terminal
	if os.Getenv("NO_COLOR") != "" {
		colorsEnabled = false
	}
}

// Color helper functions
func colorize(color, text string) string {
	if !colorsEnabled {
		return text
	}
	return color + text + ansiReset
}

func colorRed(text string) string    { return colorize(ansiRed, text) }
func colorGreen(text string) string  { return colorize(ansiGreen, text) }
func colorYellow(text string) string { return colorize(ansiYellow, text) }
func colorBlue(text string) string   { return colorize(ansiBlue, text) }
func colorCyan(text string) string   { return colorize(ansiCyan, text) }
func colorBold(text string) string   { return colorize(ansiBold, text) }
func colorDim(text string) string    { return colorize(ansiDim, text) }

// Output helpers
func printSuccess(message string) {
	fmt.Println(colorGreen("✓") + " " + message)
}

func printError(message string) {
	fmt.Fprintln(os.Stderr, colorRed("✗")+" "+message)
}

func printWarning(message string) {
	fmt.Println(colorYellow("⚠") + " " + message)
}

func printInfo(message string) {
	fmt.Println(colorBlue("ℹ") + " " + message)
}

func printStep(step int, total int, message string) {
	fmt.Printf("[%s/%d] %s\n", colorCyan(fmt.Sprintf("%d", step)), total, message)
}

func printHeader(title string) {
	fmt.Println("\n" + colorBold(colorCyan(title)))
	fmt.Println(colorDim("────────────────────────────────────────"))
}

func printTable(headers []string, rows [][]string) {
	// Calculate column widths
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	// Print header
	for i, h := range headers {
		fmt.Printf("%-*s  ", widths[i], colorBold(h))
	}
	fmt.Println()

	// Print separator
	for _, w := range widths {
		for j := 0; j < w; j++ {
			fmt.Print("─")
		}
		fmt.Print("  ")
	}
	fmt.Println()

	// Print rows
	for _, row := range rows {
		for i, cell := range row {
			fmt.Printf("%-*s  ", widths[i], cell)
		}
		fmt.Println()
	}
}
