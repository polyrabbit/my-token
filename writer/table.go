package writer

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/polyrabbit/token-ticker/exchange/model"

	"github.com/fatih/color"
	"github.com/gosuri/uilive"
	"github.com/mattn/go-colorable"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/viper"
)

const (
	colSymbol       = "Symbol"
	colPrice        = "Price"
	colChange1hPct  = "%Change(1h)"
	colChange24hPct = "%Change(24h)"
	colSource       = "Source"
	colUpdated      = "Updated"
)

func GetColumns() []string {
	return []string{colSymbol, colPrice, colChange1hPct, colChange24hPct, colSource, colUpdated}
}

var faint = color.New(color.Faint).SprintFunc()

type tableWriter struct {
	*uilive.Writer
	table *tablewriter.Table
}

// Set up ascii table writer
func NewTableWriter() *tableWriter {
	tw := &tableWriter{Writer: uilive.New()}
	tw.Writer.Out = colorable.NewColorableStdout() // For Windows
	tw.table = tablewriter.NewWriter(tw.Writer)
	tw.table.SetAutoFormatHeaders(false)
	tw.table.SetAutoWrapText(false)
	headers := viper.GetStringSlice("show")
	formattedHeaders := make([]string, len(headers))
	for i, hdr := range headers {
		formattedHeaders[i] = color.YellowString(hdr)
	}
	tw.table.SetHeader(formattedHeaders)
	tw.table.SetRowLine(true)
	tw.table.SetCenterSeparator(faint("-"))
	tw.table.SetColumnSeparator(faint("|"))
	tw.table.SetRowSeparator(faint("-"))
	return tw
}

func (tw *tableWriter) highlightChange(changePct float64) string {
	if changePct == math.MaxFloat64 {
		return ""
	}
	changeText := strconv.FormatFloat(changePct, 'f', 2, 64)
	if changePct == 0 {
		changeText = faint("0")
	} else if changePct > 0 {
		changeText = color.GreenString(changeText)
	} else {
		changeText = color.RedString(changeText)
	}
	return changeText
}

func (tw *tableWriter) Render(symbolPriceList []*model.SymbolPrice) {
	tw.table.ClearRows()
	// Fill in data
	for _, sp := range symbolPriceList {
		var columns []string
		for _, hdr := range viper.GetStringSlice("show") {
			switch strings.ToLower(hdr) {
			case strings.ToLower(colSymbol):
				columns = append(columns, sp.Symbol)
			case strings.ToLower(colPrice):
				columns = append(columns, sp.Price)
			case strings.ToLower(colChange1hPct):
				columns = append(columns, tw.highlightChange(sp.PercentChange1h))
			case strings.ToLower(colChange24hPct):
				columns = append(columns, tw.highlightChange(sp.PercentChange24h))
			case strings.ToLower(colSource):
				columns = append(columns, sp.Source)
			case strings.ToLower(colUpdated):
				columns = append(columns, sp.UpdateAt.Local().Format("15:04:05"))
			default:
				fmt.Fprintf(os.Stderr, "Unknown column: %s\n", hdr)
				os.Exit(1)
			}

		}
		tw.table.Append(columns)
	}

	tw.table.Render()
	tw.Flush()
}
