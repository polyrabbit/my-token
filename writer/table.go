package writer

import (
    "fmt"
    "math"
    "os"
    "strconv"
    "strings"

    "github.com/fatih/color"
    "github.com/gosuri/uilive"
    "github.com/mattn/go-colorable"
    "github.com/olekukonko/tablewriter"
    "github.com/polyrabbit/my-token/config"
    "github.com/polyrabbit/my-token/exchange"
)

var faint = color.New(color.Faint).SprintFunc()

type tableWriter struct {
    *uilive.Writer
    table       *tablewriter.Table
    columnNames []string
}

// Set up ascii table writer
func NewTableWriter(cfg *config.Config) *tableWriter {
    tw := &tableWriter{Writer: uilive.New(), columnNames: cfg.Columns}
    tw.Writer.Out = colorable.NewColorableStdout() // For Windows
    tw.table = tablewriter.NewWriter(tw.Writer)
    tw.table.SetAutoFormatHeaders(false)
    tw.table.SetAutoWrapText(false)
    formattedHeaders := make([]string, len(cfg.Columns))
    for i, column := range cfg.Columns {
        formattedHeaders[i] = color.YellowString(column)
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

func (tw *tableWriter) Render(symbolPriceList []*exchange.SymbolPrice) {
    tw.table.ClearRows()
    // Fill in data
    for _, sp := range symbolPriceList {
        var columns []string
        for _, name := range tw.columnNames {
            switch strings.ToLower(name) {
            case strings.ToLower(config.ColumnSymbol):
                columns = append(columns, sp.Symbol)
            case strings.ToLower(config.ColumnPrice):
                columns = append(columns, sp.Price)
            case strings.ToLower(config.ColumnChange1hPct):
                columns = append(columns, tw.highlightChange(sp.PercentChange1h))
            case strings.ToLower(config.ColumnChange24hPct):
                columns = append(columns, tw.highlightChange(sp.PercentChange24h))
            case strings.ToLower(config.ColumnSource):
                columns = append(columns, sp.Source)
            case strings.ToLower(config.ColumnUpdated):
                columns = append(columns, sp.UpdateAt.Local().Format("15:04:05"))
            default:
                fmt.Fprintf(os.Stderr, "Unknown column: %q\n", name)
                os.Exit(1)
            }

        }
        tw.table.Append(columns)
    }

    tw.table.Render()
    tw.Flush()
}
