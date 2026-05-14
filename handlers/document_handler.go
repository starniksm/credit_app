package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"credit_app/models"

	"github.com/gin-gonic/gin"
	"github.com/jung-kurt/gofpdf"
	"gorm.io/gorm"
)

type DocumentHandler struct {
	db *gorm.DB
}

func NewDocumentHandler(db *gorm.DB) *DocumentHandler {
	return &DocumentHandler{db: db}
}

// GenerateContractPDF generates a credit contract PDF with proper Cyrillic support
func (h *DocumentHandler) GenerateContractPDF(c *gin.Context) {
	appID := c.Param("id")

	var app models.CreditApplication
	if err := h.db.First(&app, "id = ?", appID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Заявка не найдена"})
		return
	}

	contractNumber := c.DefaultQuery("contract_number", appID)
	contractDate := c.DefaultQuery("contract_date", time.Now().Format("02.01.2006"))
	loanAmount := c.DefaultQuery("loan_amount", fmt.Sprintf("%.2f", app.RequestedAmount))
	loanTerm := c.DefaultQuery("loan_term", strconv.Itoa(app.CreditTerm))
	interestRate := c.DefaultQuery("interest_rate", "14")

	today, _ := time.Parse("02.01.2006", contractDate)
	transferDate := today.AddDate(0, 0, 4)
	repayDate := today.AddDate(0, int(parseInt(loanTerm)), 3)
	repayDateStr := repayDate.Format("02.01.2006")
	transferDateStr := transferDate.Format("02.01.2006")

	// Get font directory
	fontDir := getFontDir()

	// Debug logging
	fmt.Printf("[PDF DEBUG] Font directory: %s\n", fontDir)
	wd, _ := os.Getwd()
	fmt.Printf("[PDF DEBUG] CWD: %s\n", wd)

	// Verify font files exist
	ttfPath := filepath.Join(fontDir, "DejaVuSans.ttf")
	ttfBoldPath := filepath.Join(fontDir, "DejaVuSans-Bold.ttf")

	fmt.Printf("[PDF DEBUG] Checking TTF: %s\n", ttfPath)
	if _, err := os.Stat(ttfPath); os.IsNotExist(err) {
		// Try to list what's in the font directory for debugging
		if entries, err := os.ReadDir(fontDir); err == nil {
			fmt.Printf("[PDF DEBUG] Contents of font dir:\n")
			for _, e := range entries {
				fmt.Printf("[PDF DEBUG]   %s\n", e.Name())
			}
		} else {
			fmt.Printf("[PDF DEBUG] Cannot read font dir: %v\n", err)
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Файл шрифта не найден. Ожидаемый путь: %s", ttfPath)})
		return
	}
	if _, err := os.Stat(ttfBoldPath); os.IsNotExist(err) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Файл шрифта Bold не найден. Ожидаемый путь: %s", ttfBoldPath)})
		return
	}

	// Create PDF with font directory set
	pdf := gofpdf.New("P", "mm", "A4", fontDir)
	pdf.SetMargins(20, 15, 20)
	pdf.AddPage()

	// Load UTF-8 Cyrillic font - use just filename since fontDir is set in New()
	pdf.AddUTF8Font("dejavu", "", "DejaVuSans.ttf")
	pdf.AddUTF8Font("dejavu", "B", "DejaVuSans-Bold.ttf")

	// Check for font loading errors
	if err := pdf.Error(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка загрузки шрифта: " + err.Error()})
		return
	}

	// Build PDF content with proper Russian text
	h.buildContractPDF(pdf, app, contractNumber, contractDate, loanAmount, loanTerm, interestRate, repayDateStr, transferDateStr)

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=contract_%s.pdf", appID))
	c.Header("Content-Transfer-Encoding", "binary")

	pdf.Output(c.Writer)
}

func (h *DocumentHandler) buildContractPDF(pdf *gofpdf.Fpdf, app models.CreditApplication, contractNumber, contractDate, loanAmount, loanTerm, interestRate, repayDateStr, transferDateStr string) {
	setFont := func(style string, size float64) {
		pdf.SetFont("dejavu", style, size)
	}

	// Title
	setFont("B", 14)
	pdf.Cell(170, 10, fmt.Sprintf("Кредитный договор № %s", contractNumber))
	pdf.Ln(12)

	// City and date
	setFont("", 10)
	pdf.Cell(85, 6, "г. Москва")
	pdf.Cell(85, 6, contractDate)
	pdf.Ln(8)

	// Preamble
	setFont("", 11)
	pdf.SetLeftMargin(20)

	clientName := app.ClientName
	if clientName == "" {
		clientName = app.FullName
	}
	if clientName == "" {
		clientName = "_______________"
	}

	preamble := fmt.Sprintf(
		"АО «ТБанк» (далее — Кредитор) в лице генерального директора Близнюка Станислава Викторовича, действующего на основании Устава, с одной стороны и %s (далее — Заемщик) в лице %s, действующего на основании паспорта, с другой стороны, совместно именуемые «Стороны», заключили Договор о нижеследующем.",
		clientName, clientName,
	)
	pdf.MultiCell(0, 5, preamble, "", "J", false)
	pdf.Ln(3)

	// Section 1
	setFont("B", 12)
	pdf.Cell(170, 8, "1. Предмет Договора")
	pdf.Ln(7)

	setFont("", 11)
	amountFormatted := formatAmountClean(loanAmount)

	pdf.MultiCell(0, 5, fmt.Sprintf("1.1. Кредитор до %s передает Заемщику %s руб. (далее — Кредит), а Заемщик обязуется вернуть Кредитору Кредит и уплатить проценты по нему в порядке, установленном Договором, и в сроки, установленные Договором.", transferDateStr, amountFormatted), "", "J", false)
	pdf.Ln(2)
	pdf.MultiCell(0, 5, fmt.Sprintf("1.2. Заемщик выплачивает Кредитору проценты за пользование Кредитом — %s%% годовых.", interestRate), "", "J", false)
	pdf.Ln(2)
	pdf.MultiCell(0, 5, "1.3. Кредитор по своему выбору передает Кредит наличными или переводит на расчетный счет Заемщика. В последнем случае в назначении платежа Кредитор указывает дату и номер Договора.", "", "J", false)
	pdf.Ln(3)

	// Section 2
	setFont("B", 12)
	pdf.Cell(170, 8, "2. Срок действия Договора")
	pdf.Ln(7)

	setFont("", 11)
	pdf.MultiCell(0, 5, "2.1. Договор вступает в силу со дня его подписания обеими сторонами и действует до полного выполнения ими обязательств по Договору.", "", "J", false)
	pdf.Ln(2)
	pdf.MultiCell(0, 5, "2.2. Любая из Сторон вправе в одностороннем порядке расторгнуть Договор по письменному требованию. Об этом нужно письменно уведомить другую Сторону не менее чем за один месяц до даты расторжения Договора.", "", "J", false)
	pdf.Ln(2)
	pdf.MultiCell(0, 5, "2.3. При одностороннем отказе от исполнения Договора Сторона, которая заявила об этом, возмещает другой Стороне убытки, вызванные расторжением.", "", "J", false)
	pdf.Ln(3)

	// Section 3
	setFont("B", 12)
	pdf.Cell(170, 8, "3. Порядок расчетов")
	pdf.Ln(7)

	setFont("", 11)
	pdf.MultiCell(0, 5, fmt.Sprintf("3.1. Заемщик возвращает Кредит до %s Кредитору в той форме, в которой получил его.", repayDateStr), "", "J", false)
	pdf.Ln(2)
	pdf.MultiCell(0, 5, fmt.Sprintf("3.2. Заемщик ежемесячно не позднее 10-го числа каждого месяца перечисляет Кредитору платеж по Кредиту в суммах, указанных в приложении № 1 к Договору. Срок кредитования составляет %s месяцев.", loanTerm), "", "J", false)
	pdf.Ln(2)
	pdf.MultiCell(0, 5, "3.3. Кредит может быть возвращен досрочно полностью или по частям. Заемщик письменно уведомляет Кредитора минимум за 10 дней о досрочном погашении Кредита и выплачивает не менее 20%% от оставшейся суммы Кредита.", "", "J", false)
	pdf.Ln(2)
	pdf.MultiCell(0, 5, "3.4. При безналичном переводе Заемщик указывает в назначении платежа дату и номер Договора. В этом случае Кредит считается возвращенным в день зачисления денег на расчетный счет Кредитора.", "", "J", false)
	pdf.Ln(3)

	// Section 4
	setFont("B", 12)
	pdf.Cell(170, 8, "4. Ответственность Сторон")
	pdf.Ln(7)

	setFont("", 11)
	pdf.MultiCell(0, 5, "4.1. За неисполнение или ненадлежащее исполнение обязательств по настоящему Договору Стороны несут ответственность в соответствии с действующим законодательством Российской Федерации.", "", "J", false)
	pdf.Ln(2)
	pdf.MultiCell(0, 5, "4.2. В случае просрочки исполнения Заемщиком обязательств по возврату Кредита и/или уплате процентов, Кредитор вправе потребовать уплаты неустойки (пени) в размере 0,1%% от суммы просроченного платежа за каждый день просрочки.", "", "J", false)
	pdf.Ln(2)
	pdf.MultiCell(0, 5, "4.3. Кредитор не несет ответственности за убытки, понесенные Заемщиком в связи с неисполнением последним своих обязательств по Договору.", "", "J", false)
	pdf.Ln(3)

	// Section 5
	setFont("B", 12)
	pdf.Cell(170, 8, "5. Форс-мажор")
	pdf.Ln(7)

	setFont("", 11)
	pdf.MultiCell(0, 5, "5.1. Стороны освобождаются от ответственности за частичное или полное неисполнение своих обязательств по Договору, если такое неисполнение явилось следствием обстоятельств непреодолимой силы (форс-мажор), возникших после заключения Договора.", "", "J", false)
	pdf.Ln(2)
	pdf.MultiCell(0, 5, "5.2. Сторона, для которой создалась невозможность исполнения обязательств, должна немедленно известить другую Сторону о наступлении и прекращении указанных обстоятельств.", "", "J", false)
	pdf.Ln(3)

	// Section 6
	setFont("B", 12)
	pdf.Cell(170, 8, "6. Разрешение споров")
	pdf.Ln(7)

	setFont("", 11)
	pdf.MultiCell(0, 5, "6.1. Все споры и разногласия, возникающие между Сторонами в связи с исполнением обязательств по Договору, разрешаются путем переговоров.", "", "J", false)
	pdf.Ln(2)
	pdf.MultiCell(0, 5, "6.2. В случае невозможности урегулирования споров путем переговоров, они подлежат разрешению в Арбитражном суде города Москвы в соответствии с действующим законодательством Российской Федерации.", "", "J", false)
	pdf.Ln(3)

	// Section 7
	setFont("B", 12)
	pdf.Cell(170, 8, "7. Заключительные положения")
	pdf.Ln(7)

	setFont("", 11)
	pdf.MultiCell(0, 5, "7.1. Договор составлен в двух экземплярах, имеющих одинаковую юридическую силу, по одному для каждой из Сторон.", "", "J", false)
	pdf.Ln(2)
	pdf.MultiCell(0, 5, "7.2. Все изменения и дополнения к Договору действительны при условии, если они совершены в письменной форме и подписаны обеими Сторонами.", "", "J", false)
	pdf.Ln(2)
	pdf.MultiCell(0, 5, "7.3. Приложение № 1 (График погашения Кредита) является неотъемлемой частью Договора.", "", "J", false)
	pdf.Ln(5)

	// Section 8 - Addresses
	setFont("B", 12)
	pdf.Cell(170, 8, "8. Адреса и реквизиты Сторон")
	pdf.Ln(7)

	setFont("", 10)
	pdf.SetX(20)
	setFont("B", 10)
	pdf.Cell(80, 5, "Кредитор:")
	pdf.Ln(6)
	setFont("", 10)
	pdf.SetX(20)
	pdf.Cell(80, 4, "АО «ТБанк»")
	pdf.Ln(5)
	pdf.SetX(20)
	pdf.Cell(80, 4, "ИНН: 7710140679")
	pdf.Ln(5)
	pdf.SetX(20)
	pdf.Cell(80, 4, "КПП: 773501001")
	pdf.Ln(5)
	pdf.SetX(20)
	pdf.Cell(80, 4, "ОГРН: 1027739656008")
	pdf.Ln(5)
	pdf.SetX(20)
	pdf.MultiCell(80, 4, "127287, г. Москва, ул. Хуторская 2-я, д. 38А, стр. 26", "", "J", false)
	pdf.Ln(2)
	pdf.SetX(20)
	pdf.Cell(80, 4, "Тел.: 8 800 555-10-10")
	pdf.Ln(8)

	// Borrower column
	pdf.SetX(110)
	setFont("B", 10)
	pdf.Cell(80, 5, "Заемщик:")
	pdf.Ln(6)
	setFont("", 10)
	pdf.SetX(110)
	pdf.MultiCell(80, 4, clientName, "", "J", false)
	pdf.Ln(2)
	pdf.SetX(110)

	if app.PassportSeries != "" && app.PassportNumber != "" {
		pdf.MultiCell(80, 4, fmt.Sprintf("Паспорт: %s %s", app.PassportSeries, app.PassportNumber), "", "J", false)
	} else {
		pdf.MultiCell(80, 4, "Паспорт: _________________", "", "J", false)
	}
	pdf.Ln(2)
	pdf.SetX(110)

	registrationAddress := app.RegistrationAddress
	if registrationAddress == "" {
		registrationAddress = "_________________"
	}
	pdf.MultiCell(80, 4, fmt.Sprintf("Адрес: %s", registrationAddress), "", "J", false)
	pdf.Ln(2)
	pdf.SetX(110)

	phone := app.Phone
	if phone == "" {
		phone = "_________________"
	}
	pdf.Cell(80, 4, fmt.Sprintf("Тел.: %s", phone))
	pdf.Ln(8)

	// Signatures
	pdf.Ln(5)
	setFont("", 10)
	pdf.SetX(20)
	pdf.Cell(80, 5, "Генеральный директор:")
	pdf.SetX(110)
	pdf.Cell(80, 5, "Заемщик:")
	pdf.Ln(8)

	pdf.SetX(20)
	pdf.Cell(80, 5, "_________________ С.В. Близнюк")
	pdf.SetX(110)
	pdf.Cell(80, 5, fmt.Sprintf("_________________ %s", clientName))
	pdf.Ln(10)

	// Footer
	pdf.SetY(-15)
	setFont("", 8)
	pdf.SetTextColor(128, 128, 128)
	pdf.Cell(0, 5, "Шаблон подготовлен экспертами Бизнес-секретов")
}

// GeneratePaymentSchedulePDF generates a payment schedule PDF
func (h *DocumentHandler) GeneratePaymentSchedulePDF(c *gin.Context) {
	appID := c.Param("id")

	var app models.CreditApplication
	if err := h.db.First(&app, "id = ?", appID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Заявка не найдена"})
		return
	}

	loanAmountStr := c.DefaultQuery("loan_amount", fmt.Sprintf("%.2f", app.RequestedAmount))
	loanTermStr := c.DefaultQuery("loan_term", strconv.Itoa(app.CreditTerm))
	interestRateStr := c.DefaultQuery("interest_rate", "14")

	loanAmount := parseFloat(loanAmountStr)
	loanTerm := parseInt(loanTermStr)
	interestRate := parseFloat(interestRateStr)

	if loanAmount <= 0 {
		loanAmount = app.RequestedAmount
	}
	if loanTerm <= 0 {
		loanTerm = app.CreditTerm
	}

	monthlyRate := interestRate / 100 / 12
	monthlyPayment := loanAmount * (monthlyRate * math.Pow(1+monthlyRate, float64(loanTerm))) / (math.Pow(1+monthlyRate, float64(loanTerm)) - 1)

	// Get font directory
	fontDir := getFontDir()

	// Debug logging
	fmt.Printf("[PDF DEBUG] Font directory: %s\n", fontDir)
	wd, _ := os.Getwd()
	fmt.Printf("[PDF DEBUG] CWD: %s\n", wd)

	// Verify font files exist
	ttfPath := filepath.Join(fontDir, "DejaVuSans.ttf")
	ttfBoldPath := filepath.Join(fontDir, "DejaVuSans-Bold.ttf")

	fmt.Printf("[PDF DEBUG] Checking TTF: %s\n", ttfPath)
	if _, err := os.Stat(ttfPath); os.IsNotExist(err) {
		// Try to list what's in the font directory for debugging
		if entries, err := os.ReadDir(fontDir); err == nil {
			fmt.Printf("[PDF DEBUG] Contents of font dir:\n")
			for _, e := range entries {
				fmt.Printf("[PDF DEBUG]   %s\n", e.Name())
			}
		} else {
			fmt.Printf("[PDF DEBUG] Cannot read font dir: %v\n", err)
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Файл шрифта не найден. Ожидаемый путь: %s", ttfPath)})
		return
	}
	if _, err := os.Stat(ttfBoldPath); os.IsNotExist(err) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Файл шрифта DejaVuSans-Bold.ttf не найден"})
		return
	}

	pdf := gofpdf.New("P", "mm", "A4", fontDir)
	pdf.AddUTF8Font("dejavu", "", "DejaVuSans.ttf")
	pdf.AddUTF8Font("dejavu", "B", "DejaVuSans-Bold.ttf")

	if err := pdf.Error(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка загрузки шрифта: " + err.Error()})
		return
	}

	pdf.SetMargins(15, 15, 15)
	pdf.AddPage()

	h.buildSchedulePDF(pdf, app, appID, loanAmountStr, loanTerm, interestRateStr, loanAmount, monthlyPayment, monthlyRate)

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=schedule_%s.pdf", appID))
	c.Header("Content-Transfer-Encoding", "binary")

	pdf.Output(c.Writer)
}

func (h *DocumentHandler) buildSchedulePDF(pdf *gofpdf.Fpdf, app models.CreditApplication, appID, loanAmountStr string, loanTerm int, interestRateStr string, loanAmount, monthlyPayment, monthlyRate float64) {
	setFont := func(style string, size float64) {
		pdf.SetFont("dejavu", style, size)
	}

	setFont("B", 14)
	pdf.Cell(180, 10, "Приложение № 1")
	pdf.Ln(6)
	pdf.Cell(180, 10, fmt.Sprintf("к Кредитному договору № %s", appID))
	pdf.Ln(10)

	setFont("B", 14)
	pdf.Cell(180, 10, "График погашения Кредита")
	pdf.Ln(10)

	setFont("", 11)

	clientName := app.ClientName
	if clientName == "" {
		clientName = app.FullName
	}
	if clientName == "" {
		clientName = "_______________"
	}

	pdf.Cell(180, 6, fmt.Sprintf("Заемщик: %s", clientName))
	pdf.Ln(5)
	pdf.Cell(180, 6, fmt.Sprintf("Сумма кредита: %s руб.", formatAmountClean(loanAmountStr)))
	pdf.Ln(5)
	pdf.Cell(180, 6, fmt.Sprintf("Срок: %d месяцев", loanTerm))
	pdf.Ln(5)
	pdf.Cell(180, 6, fmt.Sprintf("Процентная ставка: %s%% годовых", interestRateStr))
	pdf.Ln(8)

	// Table header
	setFont("B", 8)
	headers := []string{"№ п/п", "Дата платежа", "Сумма платежа", "Основной долг", "Проценты", "Остаток долга"}
	widths := []float64{15, 28, 30, 28, 25, 54}

	for i, header := range headers {
		pdf.CellFormat(widths[i], 6, header, "1", 0, "C", false, 0, "")
	}
	pdf.Ln(-1)

	setFont("", 7)

	balance := loanAmount
	totalPayment := 0.0
	totalPrincipal := 0.0
	totalInterest := 0.0

	today := time.Now()
	for i := 1; i <= loanTerm; i++ {
		interest := balance * monthlyRate
		principal := monthlyPayment - interest
		balance -= principal

		paymentDate := today.AddDate(0, i, 0)
		dateStr := paymentDate.Format("02.01.2006")

		pdf.CellFormat(widths[0], 5, strconv.Itoa(i), "1", 0, "C", false, 0, "")
		pdf.CellFormat(widths[1], 5, dateStr, "1", 0, "C", false, 0, "")
		pdf.CellFormat(widths[2], 5, formatAmountClean(fmt.Sprintf("%.2f", monthlyPayment)), "1", 0, "R", false, 0, "")
		pdf.CellFormat(widths[3], 5, formatAmountClean(fmt.Sprintf("%.2f", principal)), "1", 0, "R", false, 0, "")
		pdf.CellFormat(widths[4], 5, formatAmountClean(fmt.Sprintf("%.2f", interest)), "1", 0, "R", false, 0, "")
		pdf.CellFormat(widths[5], 5, formatAmountClean(fmt.Sprintf("%.2f", balance)), "1", 0, "R", false, 0, "")
		pdf.Ln(-1)

		totalPayment += monthlyPayment
		totalPrincipal += principal
		totalInterest += interest

		if i < loanTerm && pdf.GetY() > 270 {
			pdf.AddPage()
			setFont("B", 8)
			for j, header := range headers {
				pdf.CellFormat(widths[j], 6, header, "1", 0, "C", false, 0, "")
			}
			pdf.Ln(-1)
			setFont("", 7)
		}
	}

	setFont("B", 8)
	pdf.CellFormat(widths[0]+widths[1]+widths[2], 6, "ИТОГО:", "1", 0, "R", true, 0, "")
	pdf.CellFormat(widths[3], 6, formatAmountClean(fmt.Sprintf("%.2f", totalPrincipal)), "1", 0, "R", true, 0, "")
	pdf.CellFormat(widths[4], 6, formatAmountClean(fmt.Sprintf("%.2f", totalInterest)), "1", 0, "R", true, 0, "")
	pdf.CellFormat(widths[5], 6, "-", "1", 0, "C", true, 0, "")
	pdf.Ln(-1)

	// Footer
	pdf.SetY(-10)
	setFont("", 8)
	pdf.SetTextColor(128, 128, 128)
	pdf.Cell(0, 5, "Шаблон подготовлен экспертами Бизнес-секретов")
}

// SendDocumentsToClient sends the generated contract to the client application.
func (h *DocumentHandler) SendDocumentsToClient(c *gin.Context) {
	appID := c.Param("id")

	var app models.CreditApplication
	if err := h.db.First(&app, "id = ?", appID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Заявка не найдена"})
		return
	}

	email := app.Email
	if email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email клиента не указан"})
		return
	}

	var req struct {
		ContractNumber string  `json:"contract_number"`
		ProductName    string  `json:"product_name"`
		Amount         float64 `json:"amount"`
		Term           int     `json:"term"`
		Rate           float64 `json:"rate"`
	}
	_ = c.ShouldBindJSON(&req)

	if req.ContractNumber == "" {
		req.ContractNumber = appID
	}
	if req.ProductName == "" {
		req.ProductName = app.CreditType
	}
	if req.Amount <= 0 {
		req.Amount = app.RequestedAmount
	}
	if req.Term <= 0 {
		req.Term = app.CreditTerm
	}
	if req.Rate <= 0 {
		req.Rate = 14
	}

	payload := map[string]interface{}{
		"application_id":  appID,
		"contract_number": req.ContractNumber,
		"product_name":    req.ProductName,
		"amount":          req.Amount,
		"term":            req.Term,
		"rate":            req.Rate,
	}
	if err := sendContractToClientApp(payload); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	now := time.Now()
	app.Status = "contract_sent"
	app.LastStatusChangeAt = &now
	app.AddActionToHistory("contract_sent", "Кредитный аналитик", fmt.Sprintf("Договор №%s отправлен клиенту", req.ContractNumber))
	if err := h.db.Save(&app).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось обновить статус заявки: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Договор отправлен клиенту на %s", email),
		"email":   email,
		"client":  app.ClientName,
		"status":  app.Status,
	})
}

// MarkContractSigned updates the bank-side application after the client signs the contract.
func (h *DocumentHandler) MarkContractSigned(c *gin.Context) {
	if c.GetHeader("X-Contract-Token") != contractDeliveryToken() {
		c.JSON(http.StatusForbidden, gin.H{"error": "Недействительный токен"})
		return
	}

	appID := c.Param("id")
	var app models.CreditApplication
	if err := h.db.First(&app, "id = ?", appID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Заявка не найдена"})
		return
	}

	var req struct {
		ContractNumber string `json:"contract_number"`
		SignedAt       string `json:"signed_at"`
	}
	_ = c.ShouldBindJSON(&req)

	now := time.Now()
	app.Status = "issued"
	app.LastStatusChangeAt = &now
	details := "Клиент подписал кредитный договор"
	if req.ContractNumber != "" {
		details = fmt.Sprintf("Клиент подписал договор №%s", req.ContractNumber)
	}
	app.AddActionToHistory("contract_signed", "Клиент", details)
	if err := h.db.Save(&app).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось обновить статус заявки: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Статус заявки обновлен", "status": app.Status})
}

// MarkCreditDelinquency receives delinquency facts from the client application.
func (h *DocumentHandler) MarkCreditDelinquency(c *gin.Context) {
	if c.GetHeader("X-Contract-Token") != contractDeliveryToken() {
		c.JSON(http.StatusForbidden, gin.H{"error": "Недействительный токен"})
		return
	}

	var payload map[string]interface{}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректные данные просрочки: " + err.Error()})
		return
	}
	req := struct {
		ApplicationID        string
		ActiveCredits        int
		TotalCredits         int
		TotalDebt            float64
		DelayedPayments12m   int
		CurrentDelinquencies bool
		MaxDelinquencyDays   int
		ActiveCreditsList    string
		DebtBurdenDetails    string
		DaysOverdue          int
		MissedPayments       int
		NextPaymentDate      string
	}{
		ApplicationID:        stringValue(payload["application_id"]),
		ActiveCredits:        intValue(payload["active_credits"]),
		TotalCredits:         intValue(payload["total_credits"]),
		TotalDebt:            floatValue(payload["total_debt"]),
		DelayedPayments12m:   intValue(payload["delayed_payments_12m"]),
		CurrentDelinquencies: boolValue(payload["current_delinquencies"]),
		MaxDelinquencyDays:   intValue(payload["max_delinquency_days"]),
		ActiveCreditsList:    stringValue(payload["active_credits_list"]),
		DebtBurdenDetails:    stringValue(payload["debt_burden_details"]),
		DaysOverdue:          intValue(payload["days_overdue"]),
		MissedPayments:       intValue(payload["missed_payments"]),
		NextPaymentDate:      stringValue(payload["next_payment_date"]),
	}
	if req.ApplicationID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Не передан ID заявки"})
		return
	}

	var app models.CreditApplication
	if err := h.db.First(&app, "id = ?", req.ApplicationID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Заявка не найдена"})
		return
	}

	app.ActiveCredits = req.ActiveCredits
	if req.TotalCredits > 0 {
		app.TotalCredits = req.TotalCredits
	}
	app.TotalDebt = req.TotalDebt
	app.DelayedPayments12m = req.DelayedPayments12m
	app.CurrentDelinquencies = req.CurrentDelinquencies || req.DaysOverdue > 0
	app.MaxDelinquencyDays = req.MaxDelinquencyDays
	if req.DaysOverdue > app.MaxDelinquencyDays {
		app.MaxDelinquencyDays = req.DaysOverdue
	}
	if req.MissedPayments > app.DelayedPayments12m {
		app.DelayedPayments12m = req.MissedPayments
	}
	app.ActiveCreditsList = req.ActiveCreditsList
	app.DebtBurdenDetails = req.DebtBurdenDetails
	if app.CurrentDelinquencies && app.RiskScore < 70 {
		app.RiskScore = 70
	}

	details := fmt.Sprintf("Зафиксирована просрочка по кредиту: %d дн.", app.MaxDelinquencyDays)
	if req.NextPaymentDate != "" {
		details = fmt.Sprintf("Зафиксирована просрочка по кредиту: %d дн., дата платежа %s", app.MaxDelinquencyDays, req.NextPaymentDate)
	}
	app.AddActionToHistory("credit_delinquency", "Система", details)

	if err := h.db.Save(&app).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось сохранить просрочку: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":               "Просрочка зафиксирована",
		"application_id":        app.ID,
		"current_delinquencies": app.CurrentDelinquencies,
		"max_delinquency_days":  app.MaxDelinquencyDays,
	})
}

// MarkRestructureDecision applies or logs the client decision on proposed credit terms.
func (h *DocumentHandler) MarkRestructureDecision(c *gin.Context) {
	if c.GetHeader("X-Contract-Token") != contractDeliveryToken() {
		c.JSON(http.StatusForbidden, gin.H{"error": "Недействительный токен"})
		return
	}

	var payload map[string]interface{}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректные данные решения: " + err.Error()})
		return
	}
	appID := stringValue(payload["application_id"])
	decision := stringValue(payload["decision"])
	if appID == "" || decision == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Не переданы ID заявки или решение клиента"})
		return
	}

	var app models.CreditApplication
	if err := h.db.First(&app, "id = ?", appID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Заявка не найдена"})
		return
	}

	newPayment := floatValue(payload["new_payment"])
	newTerm := intValue(payload["new_term_months"])
	graceMonths := intValue(payload["grace_months"])
	if decision == "accepted" {
		if newPayment > 0 {
			app.MonthlyPayment = newPayment
		}
		if newTerm > 0 {
			app.CreditTerm = newTerm
		}
		app.CurrentDelinquencies = false
		app.MaxDelinquencyDays = 0
		app.DelayedPayments12m = 0
		details := fmt.Sprintf("Клиент принял новые условия кредита: платеж %.0f ₽, срок %d мес., льготный период %d мес.", newPayment, newTerm, graceMonths)
		app.AddActionToHistory("restructure_accepted", "Клиент", details)
		app.Notes = strings.TrimSpace(app.Notes + "\\n" + time.Now().Format("2006-01-02 15:04") + " " + details)
	} else {
		details := "Клиент отказался от новых условий. Заявка передана в отдел взысканий."
		app.AddActionToHistory("collections_transfer", "Система", details)
		app.Notes = strings.TrimSpace(app.Notes + "\\n" + time.Now().Format("2006-01-02 15:04") + " " + details)
	}

	if err := h.db.Save(&app).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось сохранить решение: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Решение клиента зафиксировано", "application_id": app.ID, "decision": decision})
}

func stringValue(value interface{}) string {
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}

func intValue(value interface{}) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		parsed, _ := strconv.Atoi(v)
		return parsed
	default:
		return 0
	}
}

func floatValue(value interface{}) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case string:
		parsed, _ := strconv.ParseFloat(v, 64)
		return parsed
	default:
		return 0
	}
}

func boolValue(value interface{}) bool {
	switch v := value.(type) {
	case bool:
		return v
	case string:
		parsed, _ := strconv.ParseBool(v)
		return parsed
	default:
		return false
	}
}

func sendContractToClientApp(payload map[string]interface{}) error {
	baseURL := os.Getenv("CLIENT_APP_BASE_URL")
	if baseURL == "" {
		baseURL = "http://127.0.0.1:8000"
	}
	token := contractDeliveryToken()

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	url := strings.TrimRight(baseURL, "/") + "/api/contracts/receive/"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Contract-Token", token)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("клиентское приложение недоступно: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("клиентское приложение вернуло HTTP %d", resp.StatusCode)
	}
	return nil
}

func contractDeliveryToken() string {
	token := os.Getenv("CONTRACT_DELIVERY_TOKEN")
	if token == "" {
		return "dev-contract-token"
	}
	return token
}

// getFontDir returns the absolute path to the fonts directory
func getFontDir() string {
	// Method 1: Use runtime.Caller to find fonts relative to this source file
	_, filename, _, ok := runtime.Caller(0)
	if ok {
		// This file is in handlers/document_handler.go
		// fonts/ is in the project root, so go up one level from handlers/
		sourceDir := filepath.Dir(filename)
		projectRoot := filepath.Dir(sourceDir)
		fontDir := filepath.Join(projectRoot, "fonts")
		if _, err := os.Stat(filepath.Join(fontDir, "DejaVuSans.ttf")); err == nil {
			return fontDir
		}
	}

	// Method 2: Check relative to executable
	execPath, err := os.Executable()
	if err == nil {
		execDir := filepath.Dir(execPath)
		fontDir := filepath.Join(execDir, "fonts")
		if _, err := os.Stat(filepath.Join(fontDir, "DejaVuSans.ttf")); err == nil {
			return fontDir
		}
		parentDir := filepath.Dir(execDir)
		fontDir = filepath.Join(parentDir, "fonts")
		if _, err := os.Stat(filepath.Join(fontDir, "DejaVuSans.ttf")); err == nil {
			return fontDir
		}
	}

	// Method 3: Check relative to current working directory
	wd, _ := os.Getwd()
	fontDir := filepath.Join(wd, "fonts")
	if _, err := os.Stat(filepath.Join(fontDir, "DejaVuSans.ttf")); err == nil {
		return fontDir
	}

	parentDir := filepath.Dir(wd)
	fontDir = filepath.Join(parentDir, "fonts")
	if _, err := os.Stat(filepath.Join(fontDir, "DejaVuSans.ttf")); err == nil {
		return fontDir
	}

	// Fallback
	return filepath.Join(wd, "fonts")
}
func formatAmountClean(amountStr string) string {
	amount := parseFloat(amountStr)
	parts := strings.Split(fmt.Sprintf("%.2f", amount), ".")
	intPart := parts[0]

	var result []string
	for len(intPart) > 3 {
		result = append([]string{intPart[len(intPart)-3:]}, result...)
		intPart = intPart[:len(intPart)-3]
	}
	if intPart != "" {
		result = append([]string{intPart}, result...)
	}

	return strings.Join(result, " ")
}

func parseFloat(s string) float64 {
	s = strings.ReplaceAll(s, " ", "")
	val, _ := strconv.ParseFloat(s, 64)
	return val
}

func parseInt(s string) int {
	val, _ := strconv.Atoi(s)
	return val
}
