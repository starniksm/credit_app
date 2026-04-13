package utils

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"credit_app/models"

	"github.com/jung-kurt/gofpdf"
	"gorm.io/gorm"
)

// ExportApplicationsToCSV exports applications to CSV format
func ExportApplicationsToCSV(db *gorm.DB, filename string) error {
	var applications []models.CreditApplication
	err := db.Find(&applications).Error
	if err != nil {
		return err
	}

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{"ID", "Full Name", "Phone", "Email", "Requested Amount", "Credit Term", "Credit Type", "Status", "Priority", "Score", "Created At", "Updated At"}
	err = writer.Write(header)
	if err != nil {
		return err
	}

	// Write data rows
	for _, app := range applications {
		row := []string{
			app.ID,
			app.FullName,
			app.Phone,
			app.Email,
			fmt.Sprintf("%.2f", app.RequestedAmount),
			strconv.Itoa(app.CreditTerm),
			app.CreditType,
			app.Status,
			app.Priority,
			fmt.Sprintf("%.2f", app.Score),
			app.CreatedAt.String(),
			app.UpdatedAt.String(),
		}
		err := writer.Write(row)
		if err != nil {
			return err
		}
	}

	return nil
}

// ExportApplicationsToPDF exports applications to PDF format
func ExportApplicationsToPDF(db *gorm.DB, filename string) error {
	var applications []models.CreditApplication
	err := db.Find(&applications).Error
	if err != nil {
		return err
	}

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(40, 10, "Credit Applications Report")

	pdf.Ln(20)
	pdf.SetFont("Arial", "B", 10)

	// Table headers
	headers := []string{"ID", "Client Name", "Amount", "Term", "Status", "Score"}
	colWidths := []float64{25, 50, 30, 20, 30, 25}

	for i, header := range headers {
		pdf.CellFormat(colWidths[i], 7, header, "1", 0, "", false, 0, "")
	}
	pdf.Ln(-1)

	pdf.SetFont("Arial", "", 8)

	// Table data
	for _, app := range applications {
		pdf.CellFormat(colWidths[0], 6, app.ID, "1", 0, "", false, 0, "")
		pdf.CellFormat(colWidths[1], 6, app.FullName, "1", 0, "", false, 0, "")
		pdf.CellFormat(colWidths[2], 6, fmt.Sprintf("%.2f", app.RequestedAmount), "1", 0, "", false, 0, "")
		pdf.CellFormat(colWidths[3], 6, strconv.Itoa(app.CreditTerm), "1", 0, "", false, 0, "")
		pdf.CellFormat(colWidths[4], 6, app.Status, "1", 0, "", false, 0, "")
		pdf.CellFormat(colWidths[5], 6, fmt.Sprintf("%.2f", app.Score), "1", 0, "", false, 0, "")
		pdf.Ln(-1)
	}

	return pdf.OutputFileAndClose(filename)
}

// ExportApplicationsToJSON exports applications to JSON format
func ExportApplicationsToJSON(db *gorm.DB, filename string) error {
	var applications []models.CreditApplication
	err := db.Find(&applications).Error
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(applications, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}
