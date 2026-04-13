package handlers

import (
	"fmt"
	"math"
	"net/http"
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

// addCyrillicFont attempts to add a Cyrillic-compatible UTF-8 font
// Returns true if the font was successfully loaded
func addCyrillicFont(pdf *gofpdf.Fpdf) (success bool) {
	success = false

	defer func() {
		if r := recover(); r != nil {
			// Font loading failed silently
			success = false
		}
	}()

	// gofpdf v1.16.2+ AddUTF8Font can auto-generate the font definition from .ttf
	// The font file must be in the font directory specified in gofpdf.New()
	pdf.AddUTF8Font("dejavu", "", "DejaVuSans.ttf")
	pdf.AddUTF8Font("dejavu", "B", "DejaVuSans-Bold.ttf")

	// Set font to verify it works - this may panic if font wasn't added
	pdf.SetFont("dejavu", "", 10)

	// Write a test cell to ensure rendering works
	_ = pdf.GetStringWidth("test")

	success = true
	return
}

// GenerateContractPDF generates a credit contract PDF matching the template
func (h *DocumentHandler) GenerateContractPDF(c *gin.Context) {
	appID := c.Param("id")

	var app models.CreditApplication
	if err := h.db.First(&app, "id = ?", appID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Заявка не найдена"})
		return
	}

	// Get parameters from query
	contractNumber := c.DefaultQuery("contract_number", appID)
	contractDate := c.DefaultQuery("contract_date", time.Now().Format("02.01.2006"))
	loanAmount := c.DefaultQuery("loan_amount", fmt.Sprintf("%.2f", app.RequestedAmount))
	loanTerm := c.DefaultQuery("loan_term", strconv.Itoa(app.CreditTerm))
	interestRate := c.DefaultQuery("interest_rate", "14")

	// Calculate dates
	today, _ := time.Parse("02.01.2006", contractDate)
	transferDate := today.AddDate(0, 0, 4)                    // 4 days from contract date
	repayDate := today.AddDate(0, int(parseInt(loanTerm)), 3) // term months + 3 days
	repayDateStr := repayDate.Format("02.01.2006")
	transferDateStr := transferDate.Format("02.01.2006")

	pdf := gofpdf.New("P", "mm", "A4", "fonts/")
	pdf.SetMargins(20, 15, 20)
	pdf.AddPage()

	// Try to add Cyrillic font
	hasCyrillic := addCyrillicFont(pdf)

	// If font failed, ensure we fall back to Arial for non-Cyrillic text
	if !hasCyrillic {
		// Set a default font to avoid "SetFont: undefined font" panic
		pdf.SetFont("Arial", "", 10)
	}

	if hasCyrillic {
		pdf.SetFont("dejavu", "", 14)
	} else {
		pdf.SetFont("Arial", "", 14)
	}

	// Title centered
	pdf.Cell(170, 10, fmt.Sprintf("Кредитный договор № %s", contractNumber))
	pdf.Ln(12)

	// City and date
	pdf.SetFont("dejavu", "", 10)
	pdf.Cell(85, 6, "г. Москва")
	pdf.Cell(85, 6, contractDate)
	pdf.Ln(8)

	// Preamble
	pdf.SetFont("dejavu", "", 11)
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
	pdf.SetFont("dejavu", "B", 12)
	pdf.Cell(170, 8, "1. Предмет Договора")
	pdf.Ln(7)

	pdf.SetFont("dejavu", "", 11)
	amountFormatted := formatAmountClean(loanAmount)

	pdf.MultiCell(0, 5, fmt.Sprintf("1.1. Кредитор до %s передает Заемщику %s ₽ (далее — Кредит), а Заемщик обязуется вернуть Кредитору Кредит и уплатить проценты по нему в порядке, установленном Договором, и в сроки, установленные Договором.", transferDateStr, amountFormatted), "", "J", false)
	pdf.Ln(2)
	pdf.MultiCell(0, 5, fmt.Sprintf("1.2. Заемщик выплачивает Кредитору проценты за пользование Кредитом — %s%% годовых.", interestRate), "", "J", false)
	pdf.Ln(2)
	pdf.MultiCell(0, 5, "1.3. Кредитор по своему выбору передает Кредит наличными или переводит на расчетный счет Заемщика. В последнем случае в назначении платежа Кредитор указывает дату и номер Договора.", "", "J", false)
	pdf.Ln(3)

	// Section 2
	pdf.SetFont("dejavu", "B", 12)
	pdf.Cell(170, 8, "2. Срок действия Договора")
	pdf.Ln(7)

	pdf.SetFont("dejavu", "", 11)
	pdf.MultiCell(0, 5, "2.1. Договор вступает в силу со дня его подписания обеими сторонами и действует до полного выполнения ими обязательств по Договору.", "", "J", false)
	pdf.Ln(2)
	pdf.MultiCell(0, 5, "2.2. Любая из Сторон вправе в одностороннем порядке расторгнуть Договор по письменному требованию. Об этом нужно письменно уведомить другую Сторону не менее чем за один месяц до даты расторжения Договора.", "", "J", false)
	pdf.Ln(2)
	pdf.MultiCell(0, 5, "2.3. При одностороннем отказе от исполнения Договора Сторона, которая заявила об этом, возмещает другой Стороне убытки, вызванные расторжением.", "", "J", false)
	pdf.Ln(3)

	// Section 3
	pdf.SetFont("dejavu", "B", 12)
	pdf.Cell(170, 8, "3. Порядок расчетов")
	pdf.Ln(7)

	pdf.SetFont("dejavu", "", 11)
	pdf.MultiCell(0, 5, fmt.Sprintf("3.1. Заемщик возвращает Кредит до %s Кредитору в той форме, в которой получил его.", repayDateStr), "", "J", false)
	pdf.Ln(2)
	pdf.MultiCell(0, 5, fmt.Sprintf("3.2. Заемщик ежемесячно не позднее 10-го числа каждого месяца перечисляет Кредитору платеж по Кредиту в суммах, указанных в приложении № 1 к Договору. Срок кредитования составляет %s месяцев.", loanTerm), "", "J", false)
	pdf.Ln(2)
	pdf.MultiCell(0, 5, "3.3. Кредит может быть возвращен досрочно полностью или по частям. Заемщик письменно уведомляет Кредитора минимум за 10 дней о досрочном погашении Кредита и выплачивает не менее 20% от оставшейся суммы Кредита.", "", "J", false)
	pdf.Ln(2)
	pdf.MultiCell(0, 5, "3.4. При безналичном переводе Заемщик указывает в назначении платежа дату и номер Договора. В этом случае Кредит считается возвращенным в день зачисления денег на расчетный счет Кредитора.", "", "J", false)
	pdf.Ln(3)

	// Section 4
	pdf.SetFont("dejavu", "B", 12)
	pdf.Cell(170, 8, "4. Ответственность Сторон")
	pdf.Ln(7)

	pdf.SetFont("dejavu", "", 11)
	pdf.MultiCell(0, 5, "4.1. За неисполнение или ненадлежащее исполнение обязательств по настоящему Договору Стороны несут ответственность в соответствии с действующим законодательством Российской Федерации.", "", "J", false)
	pdf.Ln(2)
	pdf.MultiCell(0, 5, "4.2. В случае просрочки исполнения Заемщиком обязательств по возврату Кредита и/или уплате процентов, Кредитор вправе потребовать уплаты неустойки (пени) в размере 0,1% от суммы просроченного платежа за каждый день просрочки.", "", "J", false)
	pdf.Ln(2)
	pdf.MultiCell(0, 5, "4.3. Кредитор не несет ответственности за убытки, понесенные Заемщиком в связи с неисполнением последним своих обязательств по Договору.", "", "J", false)
	pdf.Ln(3)

	// Section 5
	pdf.SetFont("dejavu", "B", 12)
	pdf.Cell(170, 8, "5. Форс-мажор")
	pdf.Ln(7)

	pdf.SetFont("dejavu", "", 11)
	pdf.MultiCell(0, 5, "5.1. Стороны освобождаются от ответственности за частичное или полное неисполнение своих обязательств по Договору, если такое неисполнение явилось следствием обстоятельств непреодолимой силы (форс-мажор), возникших после заключения Договора.", "", "J", false)
	pdf.Ln(2)
	pdf.MultiCell(0, 5, "5.2. Сторона, для которой создалась невозможность исполнения обязательств, должна немедленно известить другую Сторону о наступлении и прекращении указанных обстоятельств.", "", "J", false)
	pdf.Ln(3)

	// Section 6
	pdf.SetFont("dejavu", "B", 12)
	pdf.Cell(170, 8, "6. Разрешение споров")
	pdf.Ln(7)

	pdf.SetFont("dejavu", "", 11)
	pdf.MultiCell(0, 5, "6.1. Все споры и разногласия, возникающие между Сторонами в связи с исполнением обязательств по Договору, разрешаются путем переговоров.", "", "J", false)
	pdf.Ln(2)
	pdf.MultiCell(0, 5, "6.2. В случае невозможности урегулирования споров путем переговоров, они подлежат разрешению в Арбитражном суде города Москвы в соответствии с действующим законодательством Российской Федерации.", "", "J", false)
	pdf.Ln(3)

	// Section 7
	pdf.SetFont("dejavu", "B", 12)
	pdf.Cell(170, 8, "7. Заключительные положения")
	pdf.Ln(7)

	pdf.SetFont("dejavu", "", 11)
	pdf.MultiCell(0, 5, "7.1. Договор составлен в двух экземплярах, имеющих одинаковую юридическую силу, по одному для каждой из Сторон.", "", "J", false)
	pdf.Ln(2)
	pdf.MultiCell(0, 5, "7.2. Все изменения и дополнения к Договору действительны при условии, если они совершены в письменной форме и подписаны обеими Сторонами.", "", "J", false)
	pdf.Ln(2)
	pdf.MultiCell(0, 5, "7.3. Приложение № 1 (График погашения Кредита) является неотъемлемой частью Договора.", "", "J", false)
	pdf.Ln(5)

	// Section 8 - Addresses and Details
	pdf.SetFont("dejavu", "B", 12)
	pdf.Cell(170, 8, "8. Адреса и реквизиты Сторон")
	pdf.Ln(7)

	pdf.SetFont("dejavu", "", 10)

	// Creditor column
	pdf.SetX(20)
	pdf.SetFont("dejavu", "B", 10)
	pdf.Cell(80, 5, "Кредитор:")
	pdf.Ln(6)
	pdf.SetFont("dejavu", "", 10)
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
	pdf.SetFont("dejavu", "B", 10)
	pdf.Cell(80, 5, "Заемщик:")
	pdf.Ln(6)
	pdf.SetFont("dejavu", "", 10)
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
	pdf.SetFont("dejavu", "", 10)
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
	pdf.SetFont("dejavu", "", 8)
	pdf.SetTextColor(128, 128, 128)
	pdf.Cell(0, 5, "Шаблон подготовлен экспертами Бизнес-секретов")

	// Set response headers
	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=contract_%s.pdf", appID))
	c.Header("Content-Transfer-Encoding", "binary")

	pdf.Output(c.Writer)
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

	pdf := gofpdf.New("P", "mm", "A4", "fonts/")
	hasCyrillic := addCyrillicFont(pdf)

	pdf.SetMargins(15, 15, 15)
	pdf.AddPage()

	if hasCyrillic {
		pdf.SetFont("dejavu", "B", 14)
	} else {
		pdf.SetFont("Arial", "B", 14)
	}

	pdf.Cell(180, 10, "Приложение № 1")
	pdf.Ln(6)
	pdf.Cell(180, 10, fmt.Sprintf("к Кредитному договору № %s", appID))
	pdf.Ln(10)

	if hasCyrillic {
		pdf.SetFont("dejavu", "B", 14)
	} else {
		pdf.SetFont("Arial", "B", 14)
	}
	pdf.Cell(180, 10, "График погашения Кредита")
	pdf.Ln(10)

	if hasCyrillic {
		pdf.SetFont("dejavu", "", 11)
	} else {
		pdf.SetFont("Arial", "", 11)
	}

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
	if hasCyrillic {
		pdf.SetFont("dejavu", "B", 8)
	} else {
		pdf.SetFont("Arial", "B", 8)
	}

	headers := []string{"№ п/п", "Дата платежа", "Сумма платежа", "Основной долг", "Проценты", "Остаток долга"}
	widths := []float64{15, 30, 30, 30, 25, 50}

	for i, header := range headers {
		pdf.CellFormat(widths[i], 6, header, "1", 0, "C", false, 0, "")
	}
	pdf.Ln(-1)

	// Table data
	if hasCyrillic {
		pdf.SetFont("dejavu", "", 7)
	} else {
		pdf.SetFont("Arial", "", 7)
	}

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

		// Add new page if needed
		if i < loanTerm && pdf.GetY() > 270 {
			pdf.AddPage()
			if hasCyrillic {
				pdf.SetFont("dejavu", "B", 8)
			} else {
				pdf.SetFont("Arial", "B", 8)
			}
			for j, header := range headers {
				pdf.CellFormat(widths[j], 6, header, "1", 0, "C", false, 0, "")
			}
			pdf.Ln(-1)
			if hasCyrillic {
				pdf.SetFont("dejavu", "", 7)
			} else {
				pdf.SetFont("Arial", "", 7)
			}
		}
	}

	// Totals row
	if hasCyrillic {
		pdf.SetFont("dejavu", "B", 8)
	} else {
		pdf.SetFont("Arial", "B", 8)
	}
	pdf.CellFormat(widths[0]+widths[1]+widths[2], 6, "ИТОГО:", "1", 0, "R", true, 0, "")
	pdf.CellFormat(widths[3], 6, formatAmountClean(fmt.Sprintf("%.2f", totalPrincipal)), "1", 0, "R", true, 0, "")
	pdf.CellFormat(widths[4], 6, formatAmountClean(fmt.Sprintf("%.2f", totalInterest)), "1", 0, "R", true, 0, "")
	pdf.CellFormat(widths[5], 6, "-", "1", 0, "C", true, 0, "")
	pdf.Ln(-1)

	// Footer
	pdf.SetY(-10)
	pdf.SetFont("dejavu", "", 8)
	pdf.SetTextColor(128, 128, 128)
	pdf.Cell(0, 5, "Шаблон подготовлен экспертами Бизнес-секретов")

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=schedule_%s.pdf", appID))
	c.Header("Content-Transfer-Encoding", "binary")

	pdf.Output(c.Writer)
}

// SendDocumentsToClient sends documents to client email
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

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Документы отправлены на %s", email),
		"email":   email,
		"client":  app.ClientName,
	})
}

// Helper functions
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
