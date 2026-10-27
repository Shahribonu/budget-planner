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
		"help":             "Xarajatni yozish: <b>taksi 25 ming</b>\nBir nechtasi: <b>non 8 ming, sut 14 ming</b>\nKecha uchun: <b>kecha tushlik 50k</b>\nDaromad: <b>oylik 8 mln</b> yoki <b>+500 ming</b>\nDollarda: <b>kofe 3$</b>\n\n/today — bugungi xarajatlar\n/undo — oxirgisini bekor qilish\n/app — ilovani ochish\n/settings — kechki eslatma\n/plan — oylik reja\n/premium — Premium\n/lang — tilni o'zgartirish",

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

		"due.title":         "📅 To'lovlar va qarzlar:",
		"due.installment":   "• %s — %s, %s",
		"due.lent":          "• %s sizga %s qaytarishi kerak, %s",
		"due.borrowed":      "• %s ga %s qaytarishingiz kerak, %s",
		"due.today":         "bugun",
		"due.tomorrow":      "ertaga",
		"due.overdue":       "muddati o'tgan (%s)",
		"budget.overall":    "Oylik",
		"budget.warn":       "⚠️ %s byudjetining %d%% i ishlatildi: %s / %s.",
		"budget.over":       "🔴 %s byudjeti tugadi: %s / %s.",
		"ask.card":          "💳 HUMOCARD *%s — bu qaysi hisobingiz?",
		"btn.new_card":      "➕ Yangi: Humo *%s",
		"ask.bank_in":       "%s tushdi (%s). Bu nima?",
		"ask.bank_out":      "%s ketdi (%s). Bu nima?",
		"btn.income":        "💰 Daromad",
		"btn.expense":       "💸 Xarajat",
		"btn.own_in":        "🔁 O'z hisobimdan",
		"btn.own_out":       "🔁 O'z hisobimga",
		"ask.from":          "Qaysi hisobingizdan?",
		"ask.to":            "Qaysi hisobingizga?",
		"btn.untracked":     "Ilovada yo'q hisob",
		"bank.mismatch":     "🏦 Bankda: %s · ilovada: %s",
		"btn.reconcile":     "⚖️ Bank bo'yicha tenglash",
		"reconciled":        "⚖️ Balans bank bo'yicha tenglandi: %s",
		"plan.prompt":       "📋 %s uchun nimalarni rejalashtiryapsiz? Hammasini bitta xabarda yozing, masalan:\n<b>ijara 3 mln, kurtka 700-900 ming, AirPods 2.5 mln agar bo'lsa</b>",
		"plan.monthly":      "Oy oxiri yaqin. %s oyini rejalashtiramizmi?",
		"plan.start_btn":    "📋 Rejani boshlash",
		"plan.open_btn":     "📊 Rejani ochish",
		"plan.title":        "📋 %s rejasi",
		"plan.inputs":       "Daromad %s · majburiy %s · jamg'arma %s · kundalik %s",
		"plan.free":         "Reja uchun bo'sh pul: <b>%s</b>",
		"plan.no_income":    "Daromad hali noma'lum: ilovada oylik yoki kutilgan daromadni kiriting.",
		"plan.fits":         "✅ Sig'adi:",
		"plan.later":        "⏳ Keyinroq:",
		"plan.reconsider":   "🤔 Qayta o'ylab ko'ring:",
		"plan.r.next_month": "keyingi oy boshida",
		"plan.r.savings":    "yoki jamg'armadan bir marta %s olsangiz, shu oy",
		"plan.r.saving":     "oyiga %s jamg'arsangiz, ~%d oyda",
		"plan.r.salary":     "oylikdan oldin pul yetmaydi",
		"plan.r.shortfall":  "majburiy, lekin pul yetmaydi",
		"plan.none":         "Summalarni topa olmadim. Masalan: <b>kurtka 800 ming, kitob 100 ming</b>",
		"match.ask":         "Bu rejangizdagi «%s»mi?",
		"match.yes":         "✅ Ha",
		"match.no":          "Yo'q",
		"match.done":        "✅ «%s» rejada sotib olindi",
		"month.1":           "Yanvar", "month.2": "Fevral", "month.3": "Mart", "month.4": "Aprel", "month.5": "May", "month.6": "Iyun",
		"month.7": "Iyul", "month.8": "Avgust", "month.9": "Sentabr", "month.10": "Oktabr", "month.11": "Noyabr", "month.12": "Dekabr",
		"premium.about":       "⭐ <b>Premium</b>\n• Cheksiz byudjetlar (bepul rejada — bitta)\n• Ilovaning rivojini qo'llab-quvvatlaysiz\n\nAsosiy hisob-kitob, toifalar, qarzlar va reja doim bepul.",
		"premium.monthly_btn": "⭐ %d Stars / oy",
		"premium.yearly_btn":  "⭐ %d Stars / yil",
		"premium.active":      "⭐ Premium faol: %s gacha.",
		"premium.thanks":      "⭐ Rahmat! Premium %s gacha faol.",
		"premium.trial":       "🎁 7 xil kunda xarajat yozdingiz! Sovg'a: %s gacha Premium — cheksiz byudjetlar. /premium",
		"summary.title":       "🌙 Bugungi xulosa",
		"summary.none":        "Bugun hech narsa yozmadingiz. Xarajat bo'lgan bo'lsa, shu yerga yozing yoki ovozli xabar yuboring.",
		"summary.prompt":      "Biror narsa unutildimi? Shunchaki yozing.",
		"summary.off":         "🔕 O'chirish",
		"settings.on":         "🔔 Kechki eslatma: har kuni soat %s.",
		"settings.off_now":    "🔕 Kechki eslatma o'chirilgan.",
		"settings.turn_on":    "🔔 Yoqish",
		"settings.turn_off":   "🔕 O'chirish",
		"settings.time":       "Vaqtni o'zgartirish uchun yozing: /settings 21:30",
		"settings.bad_time":   "Vaqtni SS:DD ko'rinishida yozing, masalan /settings 21:30",
		"first_log.notify":    "Har kuni soat %s da bugungi xarajatlar xulosasini yuboraman. O'zgartirish: /settings",

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
		"help":             "Харажатни ёзиш: <b>такси 25 минг</b>\nБир нечтаси: <b>нон 8 минг, сут 14 минг</b>\nКеча учун: <b>кеча тушлик 50к</b>\nДаромад: <b>ойлик 8 млн</b> ёки <b>+500 минг</b>\nДолларда: <b>кофе 3$</b>\n\n/today — бугунги харажатлар\n/undo — охиргисини бекор қилиш\n/app — иловани очиш\n/settings — кечки эслатма\n/plan — ойлик режа\n/premium — Premium\n/lang — тилни ўзгартириш",

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

		"due.title":         "📅 Тўловлар ва қарзлар:",
		"due.installment":   "• %s — %s, %s",
		"due.lent":          "• %s сизга %s қайтариши керак, %s",
		"due.borrowed":      "• %s га %s қайтаришингиз керак, %s",
		"due.today":         "бугун",
		"due.tomorrow":      "эртага",
		"due.overdue":       "муддати ўтган (%s)",
		"budget.overall":    "Ойлик",
		"budget.warn":       "⚠️ %s бюджетининг %d%% и ишлатилди: %s / %s.",
		"budget.over":       "🔴 %s бюджети тугади: %s / %s.",
		"ask.card":          "💳 HUMOCARD *%s — бу қайси ҳисобингиз?",
		"btn.new_card":      "➕ Янги: Humo *%s",
		"ask.bank_in":       "%s тушди (%s). Бу нима?",
		"ask.bank_out":      "%s кетди (%s). Бу нима?",
		"btn.income":        "💰 Даромад",
		"btn.expense":       "💸 Харажат",
		"btn.own_in":        "🔁 Ўз ҳисобимдан",
		"btn.own_out":       "🔁 Ўз ҳисобимга",
		"ask.from":          "Қайси ҳисобингиздан?",
		"ask.to":            "Қайси ҳисобингизга?",
		"btn.untracked":     "Иловада йўқ ҳисоб",
		"bank.mismatch":     "🏦 Банкда: %s · иловада: %s",
		"btn.reconcile":     "⚖️ Банк бўйича тенглаш",
		"reconciled":        "⚖️ Баланс банк бўйича тенглади: %s",
		"plan.prompt":       "📋 %s учун нималарни режалаштиряпсиз? Ҳаммасини битта хабарда ёзинг, масалан:\n<b>ижара 3 млн, куртка 700-900 минг, AirPods 2.5 млн агар бўлса</b>",
		"plan.monthly":      "Ой охири яқин. %s ойини режалаштирамизми?",
		"plan.start_btn":    "📋 Режани бошлаш",
		"plan.open_btn":     "📊 Режани очиш",
		"plan.title":        "📋 %s режаси",
		"plan.inputs":       "Даромад %s · мажбурий %s · жамғарма %s · кундалик %s",
		"plan.free":         "Режа учун бўш пул: <b>%s</b>",
		"plan.no_income":    "Даромад ҳали номаълум: иловада ойлик ёки кутилган даромадни киритинг.",
		"plan.fits":         "✅ Сиғади:",
		"plan.later":        "⏳ Кейинроқ:",
		"plan.reconsider":   "🤔 Қайта ўйлаб кўринг:",
		"plan.r.next_month": "келаси ой бошида",
		"plan.r.savings":    "ёки жамғармадан бир марта %s олсангиз, шу ой",
		"plan.r.saving":     "ойига %s жамғарсангиз, ~%d ойда",
		"plan.r.salary":     "ойликдан олдин пул етмайди",
		"plan.r.shortfall":  "мажбурий, лекин пул етмайди",
		"plan.none":         "Суммаларни топа олмадим. Масалан: <b>куртка 800 минг, китоб 100 минг</b>",
		"match.ask":         "Бу режангиздаги «%s»ми?",
		"match.yes":         "✅ Ҳа",
		"match.no":          "Йўқ",
		"match.done":        "✅ «%s» режада сотиб олинди",
		"month.1":           "Январ", "month.2": "Феврал", "month.3": "Март", "month.4": "Апрел", "month.5": "Май", "month.6": "Июн",
		"month.7": "Июл", "month.8": "Август", "month.9": "Сентябр", "month.10": "Октябр", "month.11": "Ноябр", "month.12": "Декабр",
		"premium.about":       "⭐ <b>Premium</b>\n• Чексиз бюджетлар (бепул режада — битта)\n• Илованинг ривожини қўллаб-қувватлайсиз\n\nАсосий ҳисоб-китоб, тоифалар, қарзлар ва режа доим бепул.",
		"premium.monthly_btn": "⭐ %d Stars / ой",
		"premium.yearly_btn":  "⭐ %d Stars / йил",
		"premium.active":      "⭐ Premium фаол: %s гача.",
		"premium.thanks":      "⭐ Раҳмат! Premium %s гача фаол.",
		"premium.trial":       "🎁 7 хил кунда харажат ёздингиз! Совға: %s гача Premium — чексиз бюджетлар. /premium",
		"summary.title":       "🌙 Бугунги хулоса",
		"summary.none":        "Бугун ҳеч нарса ёзмадингиз. Харажат бўлган бўлса, шу ерга ёзинг ёки овозли хабар юборинг.",
		"summary.prompt":      "Бирор нарса унутилдими? Шунчаки ёзинг.",
		"summary.off":         "🔕 Ўчириш",
		"settings.on":         "🔔 Кечки эслатма: ҳар куни соат %s.",
		"settings.off_now":    "🔕 Кечки эслатма ўчирилган.",
		"settings.turn_on":    "🔔 Ёқиш",
		"settings.turn_off":   "🔕 Ўчириш",
		"settings.time":       "Вақтни ўзгартириш учун ёзинг: /settings 21:30",
		"settings.bad_time":   "Вақтни СС:ДД кўринишида ёзинг, масалан /settings 21:30",
		"first_log.notify":    "Ҳар куни соат %s да бугунги харажатлар хулосасини юбораман. Ўзгартириш: /settings",

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
		"help":             "Записать расход: <b>такси 25 тысяч</b>\nНесколько сразу: <b>хлеб 8 тыс, молоко 14 тыс</b>\nЗа вчера: <b>вчера обед 50к</b>\nДоход: <b>зарплата 8 млн</b> или <b>+500 тысяч</b>\nВ долларах: <b>кофе 3$</b>\n\n/today — расходы за сегодня\n/undo — отменить последнее\n/app — открыть приложение\n/settings — вечернее напоминание\n/plan — план на месяц\n/premium — Premium\n/lang — сменить язык",

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

		"due.title":         "📅 Платежи и долги:",
		"due.installment":   "• %s — %s, %s",
		"due.lent":          "• %s должен вернуть вам %s, %s",
		"due.borrowed":      "• Вернуть %s: %s, %s",
		"due.today":         "сегодня",
		"due.tomorrow":      "завтра",
		"due.overdue":       "просрочено (%s)",
		"budget.overall":    "Месячный",
		"budget.warn":       "⚠️ Бюджет «%s» израсходован на %d%%: %s из %s.",
		"budget.over":       "🔴 Бюджет «%s» исчерпан: %s из %s.",
		"ask.card":          "💳 HUMOCARD *%s — какой это счёт?",
		"btn.new_card":      "➕ Новый: Humo *%s",
		"ask.bank_in":       "Пришло %s (%s). Что это?",
		"ask.bank_out":      "Ушло %s (%s). Что это?",
		"btn.income":        "💰 Доход",
		"btn.expense":       "💸 Расход",
		"btn.own_in":        "🔁 С моего счёта",
		"btn.own_out":       "🔁 На мой счёт",
		"ask.from":          "С какого вашего счёта?",
		"ask.to":            "На какой ваш счёт?",
		"btn.untracked":     "Счёт не в приложении",
		"bank.mismatch":     "🏦 В банке: %s · в приложении: %s",
		"btn.reconcile":     "⚖️ Выровнять по банку",
		"reconciled":        "⚖️ Баланс выровнен по банку: %s",
		"plan.prompt":       "📋 Что планируете на %s? Напишите всё одним сообщением, например:\n<b>аренда 3 млн, куртка 700-900 тысяч, AirPods 2.5 млн если получится</b>",
		"plan.monthly":      "Месяц подходит к концу. Спланируем %s?",
		"plan.start_btn":    "📋 Начать план",
		"plan.open_btn":     "📊 Открыть план",
		"plan.title":        "📋 План на %s",
		"plan.inputs":       "Доход %s · обязательные %s · накопления %s · повседневные %s",
		"plan.free":         "Свободно для плана: <b>%s</b>",
		"plan.no_income":    "Доход пока неизвестен: укажите зарплату или ожидаемый доход в приложении.",
		"plan.fits":         "✅ Помещается:",
		"plan.later":        "⏳ Позже:",
		"plan.reconsider":   "🤔 Стоит подумать:",
		"plan.r.next_month": "в начале следующего месяца",
		"plan.r.savings":    "или в этом месяце, если один раз взять %s из накоплений",
		"plan.r.saving":     "если откладывать %s в месяц, через ~%d мес.",
		"plan.r.salary":     "до зарплаты не хватает денег",
		"plan.r.shortfall":  "обязательно, но денег не хватает",
		"plan.none":         "Не нашёл сумм. Например: <b>куртка 800 тысяч, книга 100 тысяч</b>",
		"match.ask":         "Это «%s» из вашего плана?",
		"match.yes":         "✅ Да",
		"match.no":          "Нет",
		"match.done":        "✅ «%s» отмечено купленным",
		"month.1":           "январь", "month.2": "февраль", "month.3": "март", "month.4": "апрель", "month.5": "май", "month.6": "июнь",
		"month.7": "июль", "month.8": "август", "month.9": "сентябрь", "month.10": "октябрь", "month.11": "ноябрь", "month.12": "декабрь",
		"premium.about":       "⭐ <b>Premium</b>\n• Неограниченные бюджеты (в бесплатном — один)\n• Поддержка развития приложения\n\nУчёт, категории, долги и план всегда бесплатны.",
		"premium.monthly_btn": "⭐ %d Stars / мес",
		"premium.yearly_btn":  "⭐ %d Stars / год",
		"premium.active":      "⭐ Premium активен до %s.",
		"premium.thanks":      "⭐ Спасибо! Premium активен до %s.",
		"premium.trial":       "🎁 Вы записывали расходы 7 разных дней! Подарок: Premium до %s — неограниченные бюджеты. /premium",
		"summary.title":       "🌙 Итоги дня",
		"summary.none":        "Сегодня вы ничего не записали. Если были расходы — напишите сюда или отправьте голосовое.",
		"summary.prompt":      "Что-то забыли? Просто напишите.",
		"summary.off":         "🔕 Выключить",
		"settings.on":         "🔔 Вечернее напоминание: каждый день в %s.",
		"settings.off_now":    "🔕 Вечернее напоминание выключено.",
		"settings.turn_on":    "🔔 Включить",
		"settings.turn_off":   "🔕 Выключить",
		"settings.time":       "Чтобы изменить время, напишите: /settings 21:30",
		"settings.bad_time":   "Укажите время в формате ЧЧ:ММ, например /settings 21:30",
		"first_log.notify":    "Каждый день в %s пришлю итоги расходов за день. Изменить: /settings",

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
		"help":             "Log a spend: <b>taxi 25k</b>\nSeveral at once: <b>bread 8k, milk 14k</b>\nFor yesterday: <b>yesterday lunch 50k</b>\nIncome: <b>salary 8 mln</b> or <b>+500k</b>\nIn dollars: <b>coffee 3$</b>\n\n/today — today's spending\n/undo — undo the last entry\n/app — open the app\n/settings — evening reminder\n/plan — monthly plan\n/premium — Premium\n/lang — change language",

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

		"due.title":         "📅 Payments and debts:",
		"due.installment":   "• %s — %s, %s",
		"due.lent":          "• %s owes you %s, %s",
		"due.borrowed":      "• You owe %s %s, %s",
		"due.today":         "due today",
		"due.tomorrow":      "due tomorrow",
		"due.overdue":       "overdue since %s",
		"budget.overall":    "Monthly",
		"budget.warn":       "⚠️ %s budget is %d%% used: %s of %s.",
		"budget.over":       "🔴 %s budget is used up: %s of %s.",
		"ask.card":          "💳 HUMOCARD *%s — which of your accounts is this?",
		"btn.new_card":      "➕ New: Humo *%s",
		"ask.bank_in":       "%s came in (%s). What is it?",
		"ask.bank_out":      "%s went out (%s). What is it?",
		"btn.income":        "💰 Income",
		"btn.expense":       "💸 Expense",
		"btn.own_in":        "🔁 From my account",
		"btn.own_out":       "🔁 To my account",
		"ask.from":          "From which of your accounts?",
		"ask.to":            "To which of your accounts?",
		"btn.untracked":     "An account not in the app",
		"bank.mismatch":     "🏦 Bank: %s · app: %s",
		"btn.reconcile":     "⚖️ Match the bank",
		"reconciled":        "⚖️ Balance now matches the bank: %s",
		"plan.prompt":       "📋 What are you planning for %s? Send it all in one message, for example:\n<b>rent 3 mln, jacket 700-900k, AirPods 2.5 mln if possible</b>",
		"plan.monthly":      "The month is nearly over. Shall we plan %s?",
		"plan.start_btn":    "📋 Start the plan",
		"plan.open_btn":     "📊 Open the plan",
		"plan.title":        "📋 %s plan",
		"plan.inputs":       "Income %s · bills %s · savings %s · everyday %s",
		"plan.free":         "Free for the plan: <b>%s</b>",
		"plan.no_income":    "Income is unknown yet: add your salary or expected income in the app.",
		"plan.fits":         "✅ Fits:",
		"plan.later":        "⏳ Later:",
		"plan.reconsider":   "🤔 Reconsider:",
		"plan.r.next_month": "early next month",
		"plan.r.savings":    "or this month if you take %s from savings once",
		"plan.r.saving":     "save %s a month, in ~%d months",
		"plan.r.salary":     "not enough money before payday",
		"plan.r.shortfall":  "a must, but money is short",
		"plan.none":         "I couldn't find amounts. For example: <b>jacket 800k, book 100k</b>",
		"match.ask":         "Is this the “%s” from your plan?",
		"match.yes":         "✅ Yes",
		"match.no":          "No",
		"match.done":        "✅ “%s” marked as bought",
		"month.1":           "January", "month.2": "February", "month.3": "March", "month.4": "April", "month.5": "May", "month.6": "June",
		"month.7": "July", "month.8": "August", "month.9": "September", "month.10": "October", "month.11": "November", "month.12": "December",
		"premium.about":       "⭐ <b>Premium</b>\n• Unlimited budgets (one on the free plan)\n• Support the app's development\n\nTracking, categories, debts and the plan are always free.",
		"premium.monthly_btn": "⭐ %d Stars / month",
		"premium.yearly_btn":  "⭐ %d Stars / year",
		"premium.active":      "⭐ Premium is active until %s.",
		"premium.thanks":      "⭐ Thank you! Premium is active until %s.",
		"premium.trial":       "🎁 You've logged spending on 7 different days! A gift: Premium until %s — unlimited budgets. /premium",
		"summary.title":       "🌙 Today's summary",
		"summary.none":        "You didn't log anything today. If you spent something, type it here or send a voice note.",
		"summary.prompt":      "Forgot something? Just type it.",
		"summary.off":         "🔕 Turn off",
		"settings.on":         "🔔 Evening reminder: every day at %s.",
		"settings.off_now":    "🔕 Evening reminder is off.",
		"settings.turn_on":    "🔔 Turn on",
		"settings.turn_off":   "🔕 Turn off",
		"settings.time":       "To change the time, send: /settings 21:30",
		"settings.bad_time":   "Use HH:MM, for example /settings 21:30",
		"first_log.notify":    "Every day at %s I'll send a summary of the day's spending. Change it: /settings",

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
