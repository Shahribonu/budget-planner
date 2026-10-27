// Package categorize guesses a category key from a short description such as
// "taksiga", "обед с коллегами" or "Korzinka". Keyword rules come first; the caller
// tries the user's learned rules and the LLM fallback when these return nothing.
package categorize

import (
	"strings"
	"unicode/utf8"

	"github.com/Shahribonu/budget-planner/backend/internal/capture/normalizer"
)

// Kind is the transaction kind a category belongs to.
type Kind string

const (
	Expense Kind = "expense"
	Income  Kind = "income"
)

type Match struct {
	Key  string
	Kind Kind
}

// keywords maps category keys to word stems in Uzbek (Latin and Cyrillic), Russian and
// English. Stems of 4+ letters match as prefixes ("taksiga", "такси"); shorter ones
// match whole words or a word plus a short case suffix ("non", "nonni", "сут").
var keywords = map[string][]string{
	// expense
	"groceries": {
		"non", "нон", "хлеб", "bread", "sut", "сут", "молок", "milk", "go'sht", "gosht", "гўшт", "мяс", "meat",
		"tuxum", "тухум", "яйц", "eggs", "sabzavot", "сабзавот", "овощ", "vegetables", "meva", "мева", "фрукт", "fruit",
		"bozor", "бозор", "рынок", "базар", "supermarket", "супермаркет", "korzinka", "корзинка", "makro", "макро",
		"havas", "хавас", "magnum", "oziq", "озиқ", "продукт", "grocer", "guruch", "гуруч", "рис", "shakar", "шакар", "сахар",
		"kartoshka", "картошк", "piyoz", "пиёз", "pomidor", "помидор", "olma", "олма", "яблок", "banan", "банан",
		"kolbasa", "колбас", "pishloq", "пишлоқ", "сыр", "qatiq", "қатиқ", "yogurt", "йогурт", "do'kon", "dokon", "дўкон", "магазин",
	},
	"cafe": {
		"kafe", "кафе", "cafe", "restoran", "ресторан", "restaurant", "tushlik", "тушлик", "обед", "lunch",
		"nonushta", "нонушта", "завтрак", "breakfast", "kechki", "ужин", "dinner", "ovqat", "овқат", "еда", "food",
		"kofe", "кофе", "coffee", "kapuchino", "капучино", "latte", "латте", "choyxona", "чойхона", "osh", "ош", "плов", "palov",
		"somsa", "сомса", "самса", "lavash", "лаваш", "burger", "бургер", "pizza", "пицц", "shashlik", "шашлик", "шашлык",
		"evos", "эвос", "kfc", "doner", "донер", "shaurma", "шаурм", "kebab", "кебаб", "wolt", "express24",
	},
	"taxi": {"taksi", "такси", "taxi", "yandex", "яндекс", "uber", "mytaxi", "bolt"},
	"transport": {
		"avtobus", "автобус", "bus", "metro", "метро", "marshrutka", "маршрут", "poyezd", "поезд", "train",
		"proezd", "проезд", "transport", "транспорт", "электричк", "afrosiyob", "афросиёб", "atto", "атто",
	},
	"fuel": {"benzin", "бензин", "petrol", "metan", "метан", "propan", "пропан", "zapravka", "заправк", "yoqilg'i", "ёқилғи", "fuel", "diesel", "дизел", "solyarka", "солярк"},
	"utilities": {
		"kommunal", "коммунал", "svet", "свет", "elektr", "электр", "gaz", "газ", "issiqlik", "иссиқлик", "отоплен",
		"kvartplata", "квартплат", "musor", "мусор", "chiqindi", "чиқинди", "utilities", "electricity",
	},
	"rent": {"ijara", "ижара", "аренд", "kvartira", "квартир", "rent"},
	"mobile_internet": {
		"telefon", "телефон", "paynet", "пейнет", "internet", "интернет", "mobil", "мобил", "beeline", "билайн",
		"ucell", "юселл", "uzmobile", "mobiuz", "мобиуз", "wifi", "связь", "aloqa", "алоқа", "phone",
	},
	"health": {
		"dori", "дори", "лекарств", "apteka", "аптек", "dorixona", "дорихона", "pharmacy", "shifokor", "шифокор", "врач", "doctor",
		"klinika", "клиник", "clinic", "stomatolog", "стоматолог", "tish", "тиш", "зуб", "dentist", "analiz", "анализ",
		"kasalxona", "касалхона", "больниц", "hospital", "medicine", "vitamin", "витамин",
	},
	"clothing": {
		"kiyim", "кийим", "одежд", "clothes", "kurtka", "куртк", "jacket", "poyabzal", "пойабзал", "обув", "shoes",
		"krossovka", "кроссовк", "ko'ylak", "кўйлак", "рубашк", "shim", "шим", "брюк", "джинс", "jinsi", "palto", "пальто",
		"futbolka", "футболк",
	},
	"education": {
		"kurs", "курс", "course", "kitob", "китоб", "книг", "book", "o'qish", "ўқиш", "учеб", "kontrakt", "контракт",
		"maktab", "мактаб", "школ", "school", "universitet", "университет", "repetitor", "репетитор", "ielts",
	},
	"entertainment": {
		"kino", "кино", "cinema", "film", "фильм", "konsert", "концерт", "teatr", "театр", "o'yin", "ўйин", "игр", "game",
		"netflix", "spotify", "youtube", "playstation", "bowling", "боулинг",
	},
	"gifts":  {"sovg'a", "sovga", "совға", "подар", "gift", "gul", "гул", "цвет", "flowers", "to'y", "тўй", "свадьб", "wedding"},
	"family": {"bola", "бола", "ребен", "дети", "детей", "детск", "bog'cha", "боғча", "садик", "детсад", "oila", "оила", "семь", "kids", "child"},
	"beauty": {
		"sartarosh", "сартарош", "парикмах", "стрижк", "salon", "салон", "manikyur", "маникюр", "kosmetik", "косметик",
		"atir", "атир", "духи", "парфюм", "barber", "барбер", "haircut",
	},
	"home": {
		"mebel", "мебел", "furniture", "idish", "идиш", "посуд", "ta'mir", "таъмир", "ремонт", "хозтовар",
		"changyutgich", "пылесос",
	},
	"travel": {
		"sayohat", "саёҳат", "путешеств", "travel", "mehmonxona", "меҳмонхона", "гостиниц", "отел", "hotel",
		"aviabilet", "авиабилет", "samolyot", "самолёт", "самолет", "flight", "viza", "виза",
	},
	"fees": {"komissiya", "комисси", "commission", "fee", "jarima", "жарима", "штраф", "fine", "процент"},
	// income
	"salary":    {"oylik", "ойлик", "maosh", "маош", "зарплат", "зп", "salary", "avans", "аванс", "payday"},
	"freelance": {"freelance", "фриланс", "loyiha", "лойиҳа", "проект", "project", "buyurtma", "буюртма", "заказ"},
	"cashback":  {"keshbek", "кешбек", "кэшбэк", "cashback"},
}

var incomeKeys = map[string]bool{"salary": true, "freelance": true, "gift_income": true, "cashback": true, "other_income": true}

// Short suffixes that may follow a short stem: nonni, osh-ga, сут-а.
var shortSuffixes = []string{"", "ga", "ka", "qa", "ni", "dan", "da", "lar", "larga", "ning", "га", "ни", "дан", "да", "лар", "а", "у", "ы", "ом", "s"}

type stem struct {
	text string
	key  string
	n    int // rune count
}

var stems []stem

func init() {
	for key, words := range keywords {
		for _, w := range words {
			stems = append(stems, stem{w, key, utf8.RuneCountInString(w)})
		}
	}
}

func matches(word string, s stem) bool {
	if s.n >= 4 {
		return strings.HasPrefix(word, s.text)
	}
	rest, ok := strings.CutPrefix(word, s.text)
	if !ok {
		return false
	}
	for _, suf := range shortSuffixes {
		if rest == suf {
			return true
		}
	}
	return false
}

// Guess returns the best keyword match for description. The longest matching stem wins
// ("nonushta" is breakfast at a cafe, not bread); income categories win over expense
// ones ("oylik keldi" is salary).
func Guess(description string) (Match, bool) {
	var best stem
	bestIncome := false
	for _, w := range normalizer.Words(description) {
		for _, s := range stems {
			if !matches(w, s) {
				continue
			}
			inc := incomeKeys[s.key]
			if best.key == "" || (inc && !bestIncome) || (inc == bestIncome && s.n > best.n) {
				best, bestIncome = s, inc
			}
		}
	}
	if best.key == "" {
		return Match{}, false
	}
	kind := Expense
	if bestIncome {
		kind = Income
	}
	return Match{Key: best.key, Kind: kind}, true
}

// Pattern is the key under which a user's choice is remembered: the description's
// words, lowercased, without numbers. "Korzinka 12" and "korzinka" share a pattern.
func Pattern(description string) string {
	return strings.Join(normalizer.Words(description), " ")
}
