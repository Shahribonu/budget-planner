// Package i18n holds bot replies in the four supported languages.
// The Mini App keeps its own catalogs in app/src/i18n; category and account type names
// here must match those.
package i18n

import (
	"fmt"
	"strconv"
	"strings"
)

const Default = "uz-Latn"

// T returns the message key in lang, falling back to Uzbek Latin, then the key itself.
// Args are applied with fmt.Sprintf.
func T(lang, key string, args ...any) string {
	s, ok := catalog[lang][key]
	if !ok {
		if s, ok = catalog[Default][key]; !ok {
			return key
		}
	}
	if len(args) > 0 {
		return fmt.Sprintf(s, args...)
	}
	return s
}

// Category returns the display name of a default category key.
func Category(lang, key string) string { return T(lang, "cat."+key) }

// AccountType returns the display name of an account type.
func AccountType(lang, typ string) string { return T(lang, "acct."+typ) }

const nnbsp = " " // narrow no-break space, the thousands separator

// Money formats minor units for display: "25 000 so'm", "$12.50", "−45 000 сум".
func Money(lang string, minor int64, currency string) string {
	neg := minor < 0
	if neg {
		minor = -minor
	}
	major, frac := minor/100, minor%100
	digits := strconv.FormatInt(major, 10)
	var b strings.Builder
	for i, r := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			b.WriteString(nnbsp)
		}
		b.WriteRune(r)
	}
	num := b.String()
	if frac != 0 {
		sep := ","
		if lang == "en" {
			sep = "."
		}
		num += fmt.Sprintf("%s%02d", sep, frac)
	}
	sign := ""
	if neg {
		sign = "−"
	}
	if currency == "USD" {
		return sign + "$" + num
	}
	return sign + num + " " + T(lang, "cur."+currency)
}

var catalog = map[string]map[string]string{
	"uz-Latn": {
		"cur.UZS": "so'm",

		"start.greeting":   "Salom, %s! Men xarajatlaringizni yozib boraman.",
		"start.try":        "Sinab ko'ring — shunchaki yozing yoki ovozli xabar yuboring:\n\n<b>tushlik 45 ming</b>\n<b>taksi 25k, non 8 ming</b>",
		"start.app_button": "📊 Ilovani ochish",
		"help":             "Xarajatni yozish: <b>taksi 25 ming</b>\nBir nechtasi: <b>non 8 ming, sut 14 ming</b>\nKecha uchun: <b>kecha tushlik 50k</b>\nDaromad: <b>oylik 8 mln</b> yoki <b>+500 ming</b>\nDollarda: <b>kofe 3$</b>\n\n/today — bugungi xarajatlar\n/undo — oxirgisini bekor qilish\n/app — ilovani ochish\n/settings — kechki eslatma\n/lang — tilni o'zgartirish",

		"card.today":       "bugun",
		"card.yesterday":   "kecha",
		"card.heard":       "🎙 «%s»",
		"card.new_account": "Yangi hisob ochildi: %s",

		"btn.edit":     "✏️ Tahrirlash",
		"btn.category": "🗂 Toifa",
		"btn.account":  "💳 Hisob",
		"btn.undo":     "↩️ Bekor qilish",
		"btn.cancel":   "✖️ Bekor",
		"btn.merge":    "Ha, o'sha",
		"btn.keep":     "Yo'q, yangi",
		"btn.back":     "← Orqaga",

		"ask.account":   "%s — %s. Qaysi hisobdan?",
		"ask.amount":    "%s uchun qancha: %s yoki %s?",
		"ask.duplicate": "%s soat %s dagi %s bilan bir xilmi?",
		"ask.category":  "Toifani tanlang:",

		"undone":         "↩️ Bekor qilindi: %s",
		"undo.nothing":   "Bekor qiladigan narsa yo'q.",
		"cancelled":      "Bekor qilindi.",
		"merged":         "Yaxshi, ikki marta yozmadim.",
		"expired":        "Bu savol eskirgan. Qaytadan yozing.",
		"not_understood": "Summani topa olmadim. Masalan: <b>taksi 25 ming</b>",
		"transfer_hint":  "O'tkazmalar va valyuta almashuvini ilovada kiriting.",
		"error":          "Nimadir xato ketdi. Birozdan keyin qayta urinib ko'ring.",
		"no_accounts":    "Hali hisob yo'q.",

		"voice.too_long":    "Ovozli xabar 60 soniyadan qisqa bo'lsin.",
		"voice.unavailable": "Ovozli xabarlarni tez orada tushunaman. Hozircha yozing: <b>taksi 25 ming</b>",
		"voice.failed":      "Ovozni tushunolmadim. Yozib yuboring: <b>taksi 25 ming</b>",

		"today.none":        "Bugun hali xarajat yozilmagan.",
		"today.summary_one": "Bugun: 1 ta xarajat, %s.",
		"lang.choose":       "Tilni tanlang:",
		"lang.set":          "Til: O'zbekcha.",
		"app.unavailable":   "Ilova hali internetga chiqarilmagan. Hozircha hammasini shu chatda yozing.",
		"today.summary":     "Bugun: %d ta xarajat, %s.",

		"summary.title":     "🌙 Bugungi xulosa",
		"summary.none":      "Bugun hech narsa yozmadingiz. Xarajat bo'lgan bo'lsa, shu yerga yozing yoki ovozli xabar yuboring.",
		"summary.prompt":    "Biror narsa unutildimi? Shunchaki yozing.",
		"summary.off":       "🔕 O'chirish",
		"settings.on":       "🔔 Kechki eslatma: har kuni soat %s.",
		"settings.off_now":  "🔕 Kechki eslatma o'chirilgan.",
		"settings.turn_on":  "🔔 Yoqish",
		"settings.turn_off": "🔕 O'chirish",
		"settings.time":     "Vaqtni o'zgartirish uchun yozing: /settings 21:30",
		"settings.bad_time": "Vaqtni SS:DD ko'rinishida yozing, masalan /settings 21:30",
		"first_log.notify":  "Har kuni soat %s da bugungi xarajatlar xulosasini yuboraman. O'zgartirish: /settings",

		"cat.groceries": "Oziq-ovqat", "cat.cafe": "Kafe", "cat.taxi": "Taksi", "cat.transport": "Transport",
		"cat.fuel": "Yoqilg'i", "cat.utilities": "Kommunal", "cat.rent": "Ijara", "cat.mobile_internet": "Aloqa va internet",
		"cat.health": "Salomatlik", "cat.clothing": "Kiyim", "cat.education": "Ta'lim", "cat.entertainment": "Ko'ngilochar",
		"cat.gifts": "Sovg'alar", "cat.family": "Oila va bolalar", "cat.beauty": "Go'zallik", "cat.home": "Uy-ro'zg'or",
		"cat.travel": "Sayohat", "cat.fees": "To'lovlar va komissiya", "cat.other_expense": "Boshqa",
		"cat.salary": "Oylik", "cat.freelance": "Frilans", "cat.gift_income": "Sovg'a", "cat.cashback": "Keshbek",
		"cat.other_income": "Boshqa daromad",

		"acct.cash": "Naqd pul", "acct.humo": "Humo", "acct.uzcard": "Uzcard", "acct.visa": "Visa",
		"acct.mastercard": "Mastercard", "acct.savings": "Jamg'arma", "acct.other": "Boshqa",
		"acct.default_cash": "Naqd",
	},
	"uz-Cyrl": {
		"cur.UZS": "сўм",

		"start.greeting":   "Салом, %s! Мен харажатларингизни ёзиб бораман.",
		"start.try":        "Синаб кўринг — шунчаки ёзинг ёки овозли хабар юборинг:\n\n<b>тушлик 45 минг</b>\n<b>такси 25к, нон 8 минг</b>",
		"start.app_button": "📊 Иловани очиш",
		"help":             "Харажатни ёзиш: <b>такси 25 минг</b>\nБир нечтаси: <b>нон 8 минг, сут 14 минг</b>\nКеча учун: <b>кеча тушлик 50к</b>\nДаромад: <b>ойлик 8 млн</b> ёки <b>+500 минг</b>\nДолларда: <b>кофе 3$</b>\n\n/today — бугунги харажатлар\n/undo — охиргисини бекор қилиш\n/app — иловани очиш\n/settings — кечки эслатма\n/lang — тилни ўзгартириш",

		"card.today":       "бугун",
		"card.yesterday":   "кеча",
		"card.heard":       "🎙 «%s»",
		"card.new_account": "Янги ҳисоб очилди: %s",

		"btn.edit":     "✏️ Таҳрирлаш",
		"btn.category": "🗂 Тоифа",
		"btn.account":  "💳 Ҳисоб",
		"btn.undo":     "↩️ Бекор қилиш",
		"btn.cancel":   "✖️ Бекор",
		"btn.merge":    "Ҳа, ўша",
		"btn.keep":     "Йўқ, янги",
		"btn.back":     "← Орқага",

		"ask.account":   "%s — %s. Қайси ҳисобдан?",
		"ask.amount":    "%s учун қанча: %s ёки %s?",
		"ask.duplicate": "%s соат %s даги %s билан бир хилми?",
		"ask.category":  "Тоифани танланг:",

		"undone":         "↩️ Бекор қилинди: %s",
		"undo.nothing":   "Бекор қиладиган нарса йўқ.",
		"cancelled":      "Бекор қилинди.",
		"merged":         "Яхши, икки марта ёзмадим.",
		"expired":        "Бу савол эскирган. Қайтадан ёзинг.",
		"not_understood": "Суммани топа олмадим. Масалан: <b>такси 25 минг</b>",
		"transfer_hint":  "Ўтказмалар ва валюта алмашувини иловада киритинг.",
		"error":          "Нимадир хато кетди. Бироздан кейин қайта уриниб кўринг.",
		"no_accounts":    "Ҳали ҳисоб йўқ.",

		"voice.too_long":    "Овозли хабар 60 сониядан қисқа бўлсин.",
		"voice.unavailable": "Овозли хабарларни тез орада тушунаман. Ҳозирча ёзинг: <b>такси 25 минг</b>",
		"voice.failed":      "Овозни тушунолмадим. Ёзиб юборинг: <b>такси 25 минг</b>",

		"today.none":        "Бугун ҳали харажат ёзилмаган.",
		"today.summary_one": "Бугун: 1 та харажат, %s.",
		"lang.choose":       "Тилни танланг:",
		"lang.set":          "Тил: Ўзбекча.",
		"app.unavailable":   "Илова ҳали интернетга чиқарилмаган. Ҳозирча ҳаммасини шу чатда ёзинг.",
		"today.summary":     "Бугун: %d та харажат, %s.",

		"summary.title":     "🌙 Бугунги хулоса",
		"summary.none":      "Бугун ҳеч нарса ёзмадингиз. Харажат бўлган бўлса, шу ерга ёзинг ёки овозли хабар юборинг.",
		"summary.prompt":    "Бирор нарса унутилдими? Шунчаки ёзинг.",
		"summary.off":       "🔕 Ўчириш",
		"settings.on":       "🔔 Кечки эслатма: ҳар куни соат %s.",
		"settings.off_now":  "🔕 Кечки эслатма ўчирилган.",
		"settings.turn_on":  "🔔 Ёқиш",
		"settings.turn_off": "🔕 Ўчириш",
		"settings.time":     "Вақтни ўзгартириш учун ёзинг: /settings 21:30",
		"settings.bad_time": "Вақтни СС:ДД кўринишида ёзинг, масалан /settings 21:30",
		"first_log.notify":  "Ҳар куни соат %s да бугунги харажатлар хулосасини юбораман. Ўзгартириш: /settings",

		"cat.groceries": "Озиқ-овқат", "cat.cafe": "Кафе", "cat.taxi": "Такси", "cat.transport": "Транспорт",
		"cat.fuel": "Ёқилғи", "cat.utilities": "Коммунал", "cat.rent": "Ижара", "cat.mobile_internet": "Алоқа ва интернет",
		"cat.health": "Саломатлик", "cat.clothing": "Кийим", "cat.education": "Таълим", "cat.entertainment": "Кўнгилочар",
		"cat.gifts": "Совғалар", "cat.family": "Оила ва болалар", "cat.beauty": "Гўзаллик", "cat.home": "Уй-рўзғор",
		"cat.travel": "Саёҳат", "cat.fees": "Тўловлар ва комиссия", "cat.other_expense": "Бошқа",
		"cat.salary": "Ойлик", "cat.freelance": "Фриланс", "cat.gift_income": "Совға", "cat.cashback": "Кешбек",
		"cat.other_income": "Бошқа даромад",

		"acct.cash": "Нақд пул", "acct.humo": "Humo", "acct.uzcard": "Uzcard", "acct.visa": "Visa",
		"acct.mastercard": "Mastercard", "acct.savings": "Жамғарма", "acct.other": "Бошқа",
		"acct.default_cash": "Нақд",
	},
	"ru": {
		"cur.UZS": "сум",

		"start.greeting":   "Привет, %s! Я буду записывать ваши расходы.",
		"start.try":        "Попробуйте — просто напишите или отправьте голосовое:\n\n<b>обед 45 тысяч</b>\n<b>такси 25к, хлеб 8 тысяч</b>",
		"start.app_button": "📊 Открыть приложение",
		"help":             "Записать расход: <b>такси 25 тысяч</b>\nНесколько сразу: <b>хлеб 8 тыс, молоко 14 тыс</b>\nЗа вчера: <b>вчера обед 50к</b>\nДоход: <b>зарплата 8 млн</b> или <b>+500 тысяч</b>\nВ долларах: <b>кофе 3$</b>\n\n/today — расходы за сегодня\n/undo — отменить последнее\n/app — открыть приложение\n/settings — вечернее напоминание\n/lang — сменить язык",

		"card.today":       "сегодня",
		"card.yesterday":   "вчера",
		"card.heard":       "🎙 «%s»",
		"card.new_account": "Создан новый счёт: %s",

		"btn.edit":     "✏️ Изменить",
		"btn.category": "🗂 Категория",
		"btn.account":  "💳 Счёт",
		"btn.undo":     "↩️ Отменить",
		"btn.cancel":   "✖️ Отмена",
		"btn.merge":    "Да, это оно",
		"btn.keep":     "Нет, новая",
		"btn.back":     "← Назад",

		"ask.account":   "%s — %s. С какого счёта?",
		"ask.amount":    "Сколько за %s: %s или %s?",
		"ask.duplicate": "%s — то же, что в %s (%s)?",
		"ask.category":  "Выберите категорию:",

		"undone":         "↩️ Отменено: %s",
		"undo.nothing":   "Нечего отменять.",
		"cancelled":      "Отменено.",
		"merged":         "Хорошо, не стал записывать дважды.",
		"expired":        "Этот вопрос устарел. Напишите заново.",
		"not_understood": "Не нашёл сумму. Например: <b>такси 25 тысяч</b>",
		"transfer_hint":  "Переводы и обмен валюты добавляйте в приложении.",
		"error":          "Что-то пошло не так. Попробуйте чуть позже.",
		"no_accounts":    "Пока нет счетов.",

		"voice.too_long":    "Голосовое должно быть короче 60 секунд.",
		"voice.unavailable": "Скоро научусь понимать голосовые. Пока напишите: <b>такси 25 тысяч</b>",
		"voice.failed":      "Не разобрал голосовое. Напишите текстом: <b>такси 25 тысяч</b>",

		"today.none":        "Сегодня расходов пока нет.",
		"today.summary_one": "Сегодня: 1 расход, %s.",
		"lang.choose":       "Выберите язык:",
		"lang.set":          "Язык: русский.",
		"app.unavailable":   "Приложение ещё не опубликовано. Пока записывайте всё в этом чате.",
		"today.summary":     "Сегодня: расходов — %d, %s.",

		"summary.title":     "🌙 Итоги дня",
		"summary.none":      "Сегодня вы ничего не записали. Если были расходы — напишите сюда или отправьте голосовое.",
		"summary.prompt":    "Что-то забыли? Просто напишите.",
		"summary.off":       "🔕 Выключить",
		"settings.on":       "🔔 Вечернее напоминание: каждый день в %s.",
		"settings.off_now":  "🔕 Вечернее напоминание выключено.",
		"settings.turn_on":  "🔔 Включить",
		"settings.turn_off": "🔕 Выключить",
		"settings.time":     "Чтобы изменить время, напишите: /settings 21:30",
		"settings.bad_time": "Укажите время в формате ЧЧ:ММ, например /settings 21:30",
		"first_log.notify":  "Каждый день в %s пришлю итоги расходов за день. Изменить: /settings",

		"cat.groceries": "Продукты", "cat.cafe": "Кафе", "cat.taxi": "Такси", "cat.transport": "Транспорт",
		"cat.fuel": "Топливо", "cat.utilities": "Коммунальные", "cat.rent": "Аренда", "cat.mobile_internet": "Связь и интернет",
		"cat.health": "Здоровье", "cat.clothing": "Одежда", "cat.education": "Образование", "cat.entertainment": "Развлечения",
		"cat.gifts": "Подарки", "cat.family": "Семья и дети", "cat.beauty": "Красота", "cat.home": "Дом",
		"cat.travel": "Путешествия", "cat.fees": "Платежи и комиссии", "cat.other_expense": "Другое",
		"cat.salary": "Зарплата", "cat.freelance": "Фриланс", "cat.gift_income": "Подарок", "cat.cashback": "Кешбэк",
		"cat.other_income": "Другой доход",

		"acct.cash": "Наличные", "acct.humo": "Humo", "acct.uzcard": "Uzcard", "acct.visa": "Visa",
		"acct.mastercard": "Mastercard", "acct.savings": "Накопления", "acct.other": "Другое",
		"acct.default_cash": "Наличные",
	},
	"en": {
		"cur.UZS": "UZS",

		"start.greeting":   "Hi, %s! I'll keep track of your spending.",
		"start.try":        "Try it — just type or send a voice note:\n\n<b>lunch 45k</b>\n<b>taxi 25k, bread 8k</b>",
		"start.app_button": "📊 Open the app",
		"help":             "Log a spend: <b>taxi 25k</b>\nSeveral at once: <b>bread 8k, milk 14k</b>\nFor yesterday: <b>yesterday lunch 50k</b>\nIncome: <b>salary 8 mln</b> or <b>+500k</b>\nIn dollars: <b>coffee 3$</b>\n\n/today — today's spending\n/undo — undo the last entry\n/app — open the app\n/settings — evening reminder\n/lang — change language",

		"card.today":       "today",
		"card.yesterday":   "yesterday",
		"card.heard":       "🎙 “%s”",
		"card.new_account": "New account created: %s",

		"btn.edit":     "✏️ Edit",
		"btn.category": "🗂 Category",
		"btn.account":  "💳 Account",
		"btn.undo":     "↩️ Undo",
		"btn.cancel":   "✖️ Cancel",
		"btn.merge":    "Yes, same",
		"btn.keep":     "No, new one",
		"btn.back":     "← Back",

		"ask.account":   "%s — %s. Which account?",
		"ask.amount":    "How much for %s: %s or %s?",
		"ask.duplicate": "%s — same as the one at %s (%s)?",
		"ask.category":  "Choose a category:",

		"undone":         "↩️ Undone: %s",
		"undo.nothing":   "Nothing to undo.",
		"cancelled":      "Cancelled.",
		"merged":         "OK, I didn't record it twice.",
		"expired":        "This question has expired. Please send it again.",
		"not_understood": "I couldn't find an amount. For example: <b>taxi 25k</b>",
		"transfer_hint":  "Add transfers and currency exchange in the app.",
		"error":          "Something went wrong. Please try again in a moment.",
		"no_accounts":    "No accounts yet.",

		"voice.too_long":    "Please keep voice notes under 60 seconds.",
		"voice.unavailable": "I'll understand voice notes soon. For now, type it: <b>taxi 25k</b>",
		"voice.failed":      "I couldn't make out that voice note. Please type it: <b>taxi 25k</b>",

		"today.none":        "No spending logged today yet.",
		"today.summary_one": "Today: 1 expense, %s.",
		"lang.choose":       "Choose a language:",
		"lang.set":          "Language: English.",
		"app.unavailable":   "The app isn't published yet. For now, log everything here in the chat.",
		"today.summary":     "Today: %d expenses, %s.",

		"summary.title":     "🌙 Today's summary",
		"summary.none":      "You didn't log anything today. If you spent something, type it here or send a voice note.",
		"summary.prompt":    "Forgot something? Just type it.",
		"summary.off":       "🔕 Turn off",
		"settings.on":       "🔔 Evening reminder: every day at %s.",
		"settings.off_now":  "🔕 Evening reminder is off.",
		"settings.turn_on":  "🔔 Turn on",
		"settings.turn_off": "🔕 Turn off",
		"settings.time":     "To change the time, send: /settings 21:30",
		"settings.bad_time": "Use HH:MM, for example /settings 21:30",
		"first_log.notify":  "Every day at %s I'll send a summary of the day's spending. Change it: /settings",

		"cat.groceries": "Groceries", "cat.cafe": "Cafes", "cat.taxi": "Taxi", "cat.transport": "Transport",
		"cat.fuel": "Fuel", "cat.utilities": "Utilities", "cat.rent": "Rent", "cat.mobile_internet": "Phone & internet",
		"cat.health": "Health", "cat.clothing": "Clothing", "cat.education": "Education", "cat.entertainment": "Entertainment",
		"cat.gifts": "Gifts", "cat.family": "Family & kids", "cat.beauty": "Beauty", "cat.home": "Home",
		"cat.travel": "Travel", "cat.fees": "Fees", "cat.other_expense": "Other",
		"cat.salary": "Salary", "cat.freelance": "Freelance", "cat.gift_income": "Gift", "cat.cashback": "Cashback",
		"cat.other_income": "Other income",

		"acct.cash": "Cash", "acct.humo": "Humo", "acct.uzcard": "Uzcard", "acct.visa": "Visa",
		"acct.mastercard": "Mastercard", "acct.savings": "Savings", "acct.other": "Other",
		"acct.default_cash": "Cash",
	},
}
