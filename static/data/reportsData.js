// Моковые данные для отчетов и аналитики
const reportsData = {
    // Статистика по заявкам за последние 6 месяцев
    monthlyStats: [
        { month: 'Октябрь', applications: 145, approved: 89, rejected: 32, processingTime: 4.2 },
        { month: 'Ноябрь', applications: 168, approved: 102, rejected: 41, processingTime: 3.8 },
        { month: 'Декабрь', applications: 192, approved: 118, rejected: 48, processingTime: 4.5 },
        { month: 'Январь', applications: 156, approved: 95, rejected: 38, processingTime: 3.9 },
        { month: 'Февраль', applications: 178, approved: 112, rejected: 42, processingTime: 4.1 },
        { month: 'Март', applications: 201, approved: 128, rejected: 51, processingTime: 3.7 }
    ],

    // Информация по аналитикам (сотрудникам)
    analysts: [
        { id: 1, name: 'Александр Петров', applications: 89, approved: 67, rejected: 15, avgTime: 3.2, rating: 4.8 },
        { id: 2, name: 'Мария Иванова', applications: 102, approved: 78, rejected: 18, avgTime: 2.9, rating: 4.9 },
        { id: 3, name: 'Дмитрий Сидоров', applications: 76, approved: 52, rejected: 14, avgTime: 4.1, rating: 4.5 },
        { id: 4, name: 'Елена Смирнова', applications: 94, approved: 71, rejected: 16, avgTime: 3.5, rating: 4.7 },
        { id: 5, name: 'Иван Козлов', applications: 68, approved: 45, rejected: 12, avgTime: 3.8, rating: 4.6 },
        { id: 6, name: 'Ольга Морозова', applications: 85, approved: 62, rejected: 17, avgTime: 3.4, rating: 4.7 },
        { id: 7, name: 'Сергей Волков', applications: 77, approved: 54, rejected: 13, avgTime: 3.9, rating: 4.4 }
    ],

    // Распределение по типам кредитов
    creditTypes: [
        { type: 'Потребительский', count: 520, amount: 78000000, percentage: 42 },
        { type: 'Автокредит', count: 285, amount: 57000000, percentage: 23 },
        { type: 'Ипотека', count: 156, amount: 234000000, percentage: 13 },
        { type: 'Кредитная карта', count: 198, amount: 19800000, percentage: 16 },
        { type: 'Бизнес-кредит', count: 78, amount: 156000000, percentage: 6 }
    ],

    // Распределение по статусам
    statusDistribution: [
        { status: 'Новые', count: 89, color: '#17a2b8' },
        { status: 'В работе', count: 156, color: '#ffc107' },
        { status: 'Одобрено', count: 614, color: '#28a745' },
        { status: 'Отклонено', count: 232, color: '#dc3545' }
    ],

    // Метрики для разных периодов
    metrics: {
        currentMonth: {
            totalApplications: 201,
            approved: 128,
            approvedAmount: 42500000,
            avgProcessingTime: 3.7,
            approvalRate: 63.7,
            changes: {
                totalApplications: 12.9,
                approved: 14.3,
                approvedAmount: 18.5,
                avgProcessingTime: -8.9,
                approvalRate: 2.1
            }
        },
        lastMonth: {
            totalApplications: 178,
            approved: 112,
            approvedAmount: 35800000,
            avgProcessingTime: 4.1,
            approvalRate: 62.9,
            changes: {
                totalApplications: -7.3,
                approved: -3.4,
                approvedAmount: -5.2,
                avgProcessingTime: 5.1,
                approvalRate: -0.8
            }
        },
        quarter: {
            totalApplications: 535,
            approved: 335,
            approvedAmount: 112300000,
            avgProcessingTime: 3.9,
            approvalRate: 62.6,
            changes: {
                totalApplications: 5.2,
                approved: 8.1,
                approvedAmount: 12.3,
                avgProcessingTime: -4.2,
                approvalRate: 1.5
            }
        },
        year: {
            totalApplications: 1040,
            approved: 644,
            approvedAmount: 361000000,
            avgProcessingTime: 4.0,
            approvalRate: 61.9,
            changes: {
                totalApplications: 15.4,
                approved: 18.2,
                approvedAmount: 22.1,
                avgProcessingTime: -6.5,
                approvalRate: 3.2
            }
        }
    },

    // Коэффициент одобрения по месяцам
    approvalRate: [
        { month: 'Октябрь', rate: 61.4 },
        { month: 'Ноябрь', rate: 60.7 },
        { month: 'Декабрь', rate: 61.5 },
        { month: 'Январь', rate: 60.9 },
        { month: 'Февраль', rate: 62.9 },
        { month: 'Март', rate: 63.7 }
    ],

    // Время обработки по месяцам
    processingTime: [
        { month: 'Октябрь', time: 4.2 },
        { month: 'Ноябрь', time: 3.8 },
        { month: 'Декабрь', time: 4.5 },
        { month: 'Январь', time: 3.9 },
        { month: 'Февраль', time: 4.1 },
        { month: 'Март', time: 3.7 }
    ],

    // Риски и проблемные заявки
    riskDistribution: [
        { level: 'Низкий', count: 680, color: '#28a745' },
        { level: 'Средний', count: 245, color: '#ffc107' },
        { level: 'Высокий', count: 89, color: '#fd7e14' },
        { level: 'Критический', count: 23, color: '#dc3545' }
    ],

    // Проблемные заявки
    problematicApplications: [
        { id: 'CR-2026-0128', client: 'Иванов А.С.', amount: 2500000, risk: 'Критический', issue: 'Подозрительная информация о доходах', date: '2026-03-10' },
        { id: 'CR-2026-0115', client: 'Петрова Е.В.', amount: 1800000, risk: 'Высокий', issue: 'Несоответствие данных', date: '2026-03-08' },
        { id: 'CR-2026-0098', client: 'Сидоров Д.М.', amount: 950000, risk: 'Высокий', issue: 'Просроченные платежи в прошлом', date: '2026-03-05' },
        { id: 'CR-2026-0087', client: 'Козлова Т.Н.', amount: 1200000, risk: 'Средний', issue: 'Неполные документы', date: '2026-03-03' },
        { id: 'CR-2026-0076', client: 'Морозов С.П.', amount: 780000, risk: 'Средний', issue: 'Высокая закредитованность', date: '2026-02-28' }
    ]
};

// Функция для получения данных по периоду
function getDataByPeriod(period) {
    switch(period) {
        case 'currentMonth':
            return reportsData.metrics.currentMonth;
        case 'lastMonth':
            return reportsData.metrics.lastMonth;
        case 'quarter':
            return reportsData.metrics.quarter;
        case 'year':
            return reportsData.metrics.year;
        default:
            return reportsData.metrics.currentMonth;
    }
}

// Экспорт для использования в браузере
if (typeof module !== 'undefined' && module.exports) {
    module.exports = { reportsData, getDataByPeriod };
}
