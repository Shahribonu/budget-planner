// Privacy policy and terms shown in the app. Generated from one source per language;
// uz-Cyrl is transliterated from uz-Latn. Bump TERMS_VERSION in backend/internal/api/privacy.go
// when the meaning changes, so every user is asked to accept again.
// Draft: have a lawyer in Uzbekistan review before public launch.

export type LegalDoc = { title: string; updated: string; sections: { heading: string; paragraphs: string[] }[] }
type Lang = 'uz-Latn' | 'uz-Cyrl' | 'ru' | 'en'

// Fill in before launch.
export const legalContacts = {
  operator: 'Budget Planner',
  contact: '/help in @YourBudgetPlannerBot',
}

function fill(s: string): string {
  // split/join rather than replaceAll: older Android WebViews lack it.
  return s.split('{operator}').join(legalContacts.operator).split('{contact}').join(legalContacts.contact)
}

function doc(title: string, sections: [string, string[]][]): LegalDoc {
  return { title, updated: '2026-10-28', sections: sections.map(([heading, ps]) => ({ heading, paragraphs: ps.map(fill) })) }
}

export const legal: Record<Lang, { privacy: LegalDoc; terms: LegalDoc }> = {
  'uz-Latn': {
    privacy: doc("Maxfiylik siyosati", [
      ["Biz kimmiz", [
        "Budget Planner — Telegram bot va mini-ilova ko'rinishidagi shaxsiy moliya hisobchisi. Ushbu siyosatda “biz” — {operator}, xizmat operatori. Biz bilan bog'lanish: {contact}.",
        "Bu yerda qaysi shaxsiy ma'lumotlarni, nima uchun qayta ishlashimiz, ular qayerda saqlanishi va siz ular bilan nima qila olishingiz yozilgan. Siyosat O'zbekiston Respublikasining “Shaxsga doir ma'lumotlar to'g'risida”gi Qonuniga muvofiq tuzilgan.",
      ]],
      ["Nimalarni yig'amiz", [
        "Telegramdan: Telegram ID, ism, username va til. Profil rasmingiz ilova ochilganda Telegramdan olinadi va bizda saqlanmaydi.",
        "Siz kiritgan ma'lumotlar: hisoblar va balanslar, summa, kategoriya va izohli daromad va xarajatlar, qarzlar, nasiyalar, byudjetlar, oylik rejalar va doimiy to'lovlar.",
        "Yuborgan bank xabarlaringiz: summa, xarid joyi va kartaning oxirgi to'rt raqamini olamiz. Kartaning to'liq raqami hech qachon saqlanmaydi.",
        "Botga yuborgan xabar va ovozli xabarlaringiz yozuv yaratish uchun qayta ishlanadi. Tasdiqlanmagan yozuv matni 7 kungacha saqlanadi, keyin o'chiriladi. Ovoz matnga aylantirilgandan keyin saqlanmaydi.",
        "Foydalanish hodisalari, masalan “ilova ochildi” yoki “tranzaksiya qo'shildi” — summa va izohlarsiz, qaysi funksiyalar foydali ekanini tushunish uchun.",
      ]],
      ["Nima uchun foydalanamiz", [
        "Xizmat ishlashi uchun: balans va tarixni saqlash, botda javob berish va siz yoqqan eslatma va xulosalarni yuborish.",
        "Umumlashtirilgan foydalanish hodisalari asosida xizmatni yaxshilash uchun. Ma'lumotlaringizni sotmaymiz va reklama uchun ishlatmaymiz.",
      ]],
      ["Yana kim qayta ishlaydi", [
        "Telegram siz va bot o'rtasida xabarlarni yetkazadi.",
        "Avtomatik qoidalar kategoriyani aniqlay olmasa, xaridning qisqa tavsifi kategoriya taklif qilish uchun Anthropic (Claude AI) ga yuborilishi mumkin. Karta va telefon raqamlari kabi uzun raqamlar avval olib tashlanadi, summalar yuborilmaydi. Anthropic bu ma'lumotlardan modellarini o'qitish uchun foydalanmaydi.",
        "Ovozni tanish yoqilgan bo'lsa, ovozli xabarlar faqat matnga aylantirish uchun nutqni tanish xizmatiga yuboriladi.",
        "Har qanday yangi xizmatdan foydalanishdan oldin bu yerda xabar beramiz.",
      ]],
      ["Ma'lumotlar qayerda saqlanadi", [
        "Ma'lumotlaringiz O'zbekiston Respublikasi hududida joylashgan serverlarda saqlanadi. Zaxira nusxalar 14 kun saqlanadi va ular ham O'zbekistonda joylashgan.",
      ]],
      ["Qancha vaqt saqlaymiz", [
        "Hisobingiz mavjud ekan. Hisob o'chirilgach, ma'lumotlar ishchi bazadan darhol o'chiriladi va 14 kun ichida zaxira nusxalardan ham yo'qoladi.",
      ]],
      ["Huquqlaringiz", [
        "Yuklab olish: Profil → Ma'lumotlar va maxfiylik → Ma'lumotlarimni yuklab olish — bot barcha ma'lumotlaringizni CSV va JSON fayllarda yuboradi.",
        "Tuzatish: istalgan yozuvni ilova yoki botda o'zgartirish yoki o'chirish mumkin.",
        "O'chirish: Profil → Ma'lumotlar va maxfiylik → Hisobni o'chirish hammasini o'chiradi.",
        "Hisobni o'chirib, roziligingizni istalgan vaqtda qaytarib olishingiz mumkin. Boshqa so'rovlar uchun biz bilan bog'laning: {contact}.",
      ]],
      ["Xavfsizlik", [
        "Mini-ilovadan kelgan har bir so'rov Telegram imzosi bilan tekshiriladi. Ulanishlar shifrlangan (HTTPS), serverlarga kirish cheklangan, loglarda summa va izohlar bo'lmaydi.",
      ]],
      ["O'zgarishlar", [
        "Siyosatni o'zgartirsak, ilova yangi versiyani ko'rsatadi va uni qayta qabul qilishni so'raydi. Yuqoridagi sana — oxirgi versiya sanasi.",
      ]],
    ]),
    terms: doc("Foydalanish shartlari", [
      ["Xizmat", [
        "Budget Planner o'z pulingizni yozib borish va rejalashtirishga yordam beradi. Bu hisobchi: u bank emas, pul saqlamaydi va o'tkazmaydi, investitsiya, soliq yoki yuridik maslahat bermaydi.",
        "Xizmat bepul. Rejalar, “sig'adi / sig'maydi” tekshiruvlari va tavsiyalar siz kiritgan ma'lumotlarga asoslangan taxminlardir. Qarorni siz qabul qilasiz.",
      ]],
      ["Hisobingiz", [
        "Hisobingiz Telegram hisobingizga bog'langan. Kiritilgan ma'lumotlar to'g'riligi va Telegramingizga kirish uchun o'zingiz javobgarsiz.",
        "Xizmatdan foydalanib, siz ushbu shartlar va Maxfiylik siyosatini qabul qilasiz.",
      ]],
      ["Halol foydalanish", [
        "Boshqalarning ma'lumotlariga kirishga, xizmatni ortiqcha yuklash yoki buzishga, uni noqonuniy maqsadlarda ishlatishga urinmang. Bunday hisoblarni cheklashimiz yoki yopishimiz mumkin.",
      ]],
      ["Mavjudlik", [
        "Xizmat ishlashi va ma'lumotlaringiz xavfsizligi uchun harakat qilamiz, lekin xizmat “boricha” taqdim etiladi. Funksiyalarni o'zgartirishimiz yoki to'xtatishimiz mumkin, imkon qadar oldindan ogohlantiramiz.",
        "Muhim ma'lumotlar nusxasini saqlang: yuklab olish istalgan vaqtda mavjud.",
      ]],
      ["Javobgarlik", [
        "Qonun ruxsat bergan doirada xizmat yordamida qabul qilingan qarorlar yoki siz kiritgan ma'lumotlardagi xatolar tufayli yuzaga kelgan zararlar uchun javobgar emasmiz.",
      ]],
      ["O'zgarishlar va qonun", [
        "Bu shartlarni yangilashimiz mumkin. Yangi versiya ilovada ko'rsatiladi va uni qabul qilish so'raladi.",
        "Shartlar O'zbekiston Respublikasi qonunchiligi bilan tartibga solinadi.",
      ]],
    ]),
  },
  'uz-Cyrl': {
    privacy: doc("Махфийлик сиёсати", [
      ["Биз киммиз", [
        "Budget Planner — Telegram бот ва мини-илова кўринишидаги шахсий молия ҳисобчиси. Ушбу сиёсатда “биз” — {operator}, хизмат оператори. Биз билан боғланиш: {contact}.",
        "Бу ерда қайси шахсий маълумотларни, нима учун қайта ишлашимиз, улар қаерда сақланиши ва сиз улар билан нима қила олишингиз ёзилган. Сиёсат Ўзбекистон Республикасининг “Шахсга доир маълумотлар тўғрисида”ги Қонунига мувофиқ тузилган.",
      ]],
      ["Нималарни йиғамиз", [
        "Telegramдан: Telegram ID, исм, username ва тил. Профил расмингиз илова очилганда Telegramдан олинади ва бизда сақланмайди.",
        "Сиз киритган маълумотлар: ҳисоблар ва баланслар, сумма, категория ва изоҳли даромад ва харажатлар, қарзлар, насиялар, бюджетлар, ойлик режалар ва доимий тўловлар.",
        "Юборган банк хабарларингиз: сумма, харид жойи ва картанинг охирги тўрт рақамини оламиз. Картанинг тўлиқ рақами ҳеч қачон сақланмайди.",
        "Ботга юборган хабар ва овозли хабарларингиз ёзув яратиш учун қайта ишланади. Тасдиқланмаган ёзув матни 7 кунгача сақланади, кейин ўчирилади. Овоз матнга айлантирилгандан кейин сақланмайди.",
        "Фойдаланиш ҳодисалари, масалан “илова очилди” ёки “транзаксия қўшилди” — сумма ва изоҳларсиз, қайси функсиялар фойдали эканини тушуниш учун.",
      ]],
      ["Нима учун фойдаланамиз", [
        "Хизмат ишлаши учун: баланс ва тарихни сақлаш, ботда жавоб бериш ва сиз ёққан эслатма ва хулосаларни юбориш.",
        "Умумлаштирилган фойдаланиш ҳодисалари асосида хизматни яхшилаш учун. Маълумотларингизни сотмаймиз ва реклама учун ишлатмаймиз.",
      ]],
      ["Яна ким қайта ишлайди", [
        "Telegram сиз ва бот ўртасида хабарларни етказади.",
        "Автоматик қоидалар категорияни аниқлай олмаса, хариднинг қисқа тавсифи категория таклиф қилиш учун Anthropic (Claude AI) га юборилиши мумкин. Карта ва телефон рақамлари каби узун рақамлар аввал олиб ташланади, суммалар юборилмайди. Anthropic бу маълумотлардан моделларини ўқитиш учун фойдаланмайди.",
        "Овозни таниш ёқилган бўлса, овозли хабарлар фақат матнга айлантириш учун нутқни таниш хизматига юборилади.",
        "Ҳар қандай янги хизматдан фойдаланишдан олдин бу ерда хабар берамиз.",
      ]],
      ["Маълумотлар қаерда сақланади", [
        "Маълумотларингиз Ўзбекистон Республикаси ҳудудида жойлашган серверларда сақланади. Захира нусхалар 14 кун сақланади ва улар ҳам Ўзбекистонда жойлашган.",
      ]],
      ["Қанча вақт сақлаймиз", [
        "Ҳисобингиз мавжуд экан. Ҳисоб ўчирилгач, маълумотлар ишчи базадан дарҳол ўчирилади ва 14 кун ичида захира нусхалардан ҳам йўқолади.",
      ]],
      ["Ҳуқуқларингиз", [
        "Юклаб олиш: Профил → Маълумотлар ва махфийлик → Маълумотларимни юклаб олиш — бот барча маълумотларингизни CSV ва JSON файлларда юборади.",
        "Тузатиш: исталган ёзувни илова ёки ботда ўзгартириш ёки ўчириш мумкин.",
        "Ўчириш: Профил → Маълумотлар ва махфийлик → Ҳисобни ўчириш ҳаммасини ўчиради.",
        "Ҳисобни ўчириб, розилигингизни исталган вақтда қайтариб олишингиз мумкин. Бошқа сўровлар учун биз билан боғланинг: {contact}.",
      ]],
      ["Хавфсизлик", [
        "Мини-иловадан келган ҳар бир сўров Telegram имзоси билан текширилади. Уланишлар шифрланган (HTTPS), серверларга кириш чекланган, логларда сумма ва изоҳлар бўлмайди.",
      ]],
      ["Ўзгаришлар", [
        "Сиёсатни ўзгартирсак, илова янги версияни кўрсатади ва уни қайта қабул қилишни сўрайди. Юқоридаги сана — охирги версия санаси.",
      ]],
    ]),
    terms: doc("Фойдаланиш шартлари", [
      ["Хизмат", [
        "Budget Planner ўз пулингизни ёзиб бориш ва режалаштиришга ёрдам беради. Бу ҳисобчи: у банк эмас, пул сақламайди ва ўтказмайди, инвеститсия, солиқ ёки юридик маслаҳат бермайди.",
        "Хизмат бепул. Режалар, “сиғади / сиғмайди” текширувлари ва тавсиялар сиз киритган маълумотларга асосланган тахминлардир. Қарорни сиз қабул қиласиз.",
      ]],
      ["Ҳисобингиз", [
        "Ҳисобингиз Telegram ҳисобингизга боғланган. Киритилган маълумотлар тўғрилиги ва Telegramингизга кириш учун ўзингиз жавобгарсиз.",
        "Хизматдан фойдаланиб, сиз ушбу шартлар ва Махфийлик сиёсатини қабул қиласиз.",
      ]],
      ["Ҳалол фойдаланиш", [
        "Бошқаларнинг маълумотларига киришга, хизматни ортиқча юклаш ёки бузишга, уни ноқонуний мақсадларда ишлатишга уринманг. Бундай ҳисобларни чеклашимиз ёки ёпишимиз мумкин.",
      ]],
      ["Мавжудлик", [
        "Хизмат ишлаши ва маълумотларингиз хавфсизлиги учун ҳаракат қиламиз, лекин хизмат “борича” тақдим этилади. Функсияларни ўзгартиришимиз ёки тўхтатишимиз мумкин, имкон қадар олдиндан огоҳлантирамиз.",
        "Муҳим маълумотлар нусхасини сақланг: юклаб олиш исталган вақтда мавжуд.",
      ]],
      ["Жавобгарлик", [
        "Қонун рухсат берган доирада хизмат ёрдамида қабул қилинган қарорлар ёки сиз киритган маълумотлардаги хатолар туфайли юзага келган зарарлар учун жавобгар эмасмиз.",
      ]],
      ["Ўзгаришлар ва қонун", [
        "Бу шартларни янгилашимиз мумкин. Янги версия иловада кўрсатилади ва уни қабул қилиш сўралади.",
        "Шартлар Ўзбекистон Республикаси қонунчилиги билан тартибга солинади.",
      ]],
    ]),
  },
  'ru': {
    privacy: doc("Политика конфиденциальности", [
      ["Кто мы", [
        "Budget Planner — личный финансовый трекер в виде Telegram-бота и мини-приложения. В этой политике «мы» — {operator}, оператор сервиса. Связаться с нами: {contact}.",
        "Здесь описано, какие персональные данные мы обрабатываем, зачем, где они хранятся и что вы можете с ними сделать. Политика составлена в соответствии с Законом Республики Узбекистан «О персональных данных».",
      ]],
      ["Что мы собираем", [
        "Из Telegram: ваш Telegram ID, имя, username и язык. Фото профиля загружается из Telegram при открытии приложения и у нас не хранится.",
        "То, что вы вводите: счета и балансы, доходы и расходы с суммами, категориями и заметками, долги, рассрочки, бюджеты, планы на месяц и регулярные платежи.",
        "Пересланные банковские уведомления: мы берём сумму, место покупки и последние четыре цифры карты. Полный номер карты никогда не сохраняется.",
        "Сообщения и голосовые, отправленные боту, обрабатываются, чтобы создать записи. Текст неподтверждённой записи хранится до 7 дней, затем удаляется. Аудио после распознавания не хранится.",
        "События использования, например «открыто приложение» или «добавлена операция», — без сумм и описаний, чтобы понимать, какие функции полезны.",
      ]],
      ["Зачем мы их используем", [
        "Чтобы сервис работал: хранить балансы и историю, отвечать в боте и присылать включённые вами напоминания и сводки.",
        "Чтобы улучшать сервис по обобщённым событиям использования. Мы не продаём ваши данные и не используем их для рекламы.",
      ]],
      ["Кто ещё их обрабатывает", [
        "Telegram доставляет сообщения между вами и ботом.",
        "Если автоматические правила не могут определить категорию, короткое описание покупки может быть отправлено в Anthropic (Claude AI) для подсказки. Длинные числа, такие как номера карт и телефонов, предварительно удаляются, суммы не отправляются. Anthropic не использует эти данные для обучения моделей.",
        "Если включено распознавание голоса, голосовые отправляются сервису распознавания речи только для расшифровки.",
        "О любом новом сервисе мы сообщим здесь до начала его использования.",
      ]],
      ["Где хранятся данные", [
        "Данные хранятся на серверах, расположенных в Республике Узбекистан. Резервные копии хранятся 14 дней и тоже находятся в Узбекистане.",
      ]],
      ["Сколько мы их храним", [
        "Пока у вас есть аккаунт. После удаления аккаунта данные сразу стираются из рабочей базы и исчезают из резервных копий в течение 14 дней.",
      ]],
      ["Ваши права", [
        "Выгрузка: Профиль → Данные и конфиденциальность → Выгрузить мои данные — бот пришлёт все ваши данные файлами CSV и JSON.",
        "Исправление: любую запись можно изменить или удалить в приложении или боте.",
        "Удаление: Профиль → Данные и конфиденциальность → Удалить аккаунт стирает всё.",
        "Вы можете отозвать согласие в любой момент, удалив аккаунт. По другим вопросам пишите нам: {contact}.",
      ]],
      ["Безопасность", [
        "Каждый запрос из мини-приложения проверяется подписью Telegram. Соединения зашифрованы (HTTPS), доступ к серверам ограничен, в логах нет сумм и описаний.",
      ]],
      ["Изменения", [
        "Если мы изменим политику, приложение покажет новую версию и попросит принять её снова. Дата вверху — дата последней версии.",
      ]],
    ]),
    terms: doc("Условия использования", [
      ["Сервис", [
        "Budget Planner помогает записывать и планировать ваши собственные деньги. Это трекер: он не является банком, не хранит и не переводит деньги и не даёт инвестиционных, налоговых или юридических консультаций.",
        "Сервис бесплатный. Планы, проверки «помещается / не помещается» и подсказки — это оценки на основе введённых вами данных. Решения принимаете вы.",
      ]],
      ["Ваш аккаунт", [
        "Аккаунт привязан к вашему аккаунту Telegram. Вы отвечаете за точность введённых данных и за доступ к своему Telegram.",
        "Пользуясь сервисом, вы принимаете эти условия и Политику конфиденциальности.",
      ]],
      ["Добросовестное использование", [
        "Не пытайтесь получить доступ к чужим данным, перегрузить или сломать сервис или использовать его для незаконных целей. Такие аккаунты мы можем ограничить или закрыть.",
      ]],
      ["Доступность", [
        "Мы стараемся, чтобы сервис работал, а данные были в сохранности, но сервис предоставляется «как есть». Мы можем менять или отключать функции и, когда возможно, предупредим заранее.",
        "Храните копию важных данных: выгрузка доступна в любое время.",
      ]],
      ["Ответственность", [
        "В пределах, допустимых законом, мы не несём ответственности за убытки от решений, принятых с помощью сервиса, или от ошибок во введённых вами данных.",
      ]],
      ["Изменения и право", [
        "Мы можем обновлять эти условия. Новая версия появится в приложении, и вас попросят её принять.",
        "Условия регулируются законодательством Республики Узбекистан.",
      ]],
    ]),
  },
  'en': {
    privacy: doc("Privacy Policy", [
      ["Who we are", [
        "Budget Planner is a personal finance tracker that works as a Telegram bot and Mini App. In this policy “we” means {operator}, the operator of the service. Contact us through {contact}.",
        "This policy explains what personal data we process, why, where it is stored and what you can do with it. It follows the Law of the Republic of Uzbekistan “On Personal Data”.",
      ]],
      ["What we collect", [
        "From Telegram: your Telegram ID, first name, username and language. Your profile photo is fetched from Telegram when you open the app and is not stored by us.",
        "What you enter: accounts and balances, income and expenses with amounts, categories and notes, debts, installments, budgets, monthly plans and recurring payments.",
        "Bank notifications you forward: we read the amount, merchant and the last four digits of the card. Full card numbers are never stored.",
        "Messages and voice notes you send to the bot are processed to create records. The text of an unconfirmed record is kept for up to 7 days, then deleted. Voice audio is not stored after it is transcribed.",
        "Usage events, such as “app opened” or “transaction added”, without amounts or descriptions, to understand which features help.",
      ]],
      ["Why we use it", [
        "To provide the service: keep your balances and history, answer in the bot and send the reminders and summaries you turn on.",
        "To improve the service using aggregated usage events. We do not sell your data and do not use it for advertising.",
      ]],
      ["Who else processes it", [
        "Telegram delivers messages between you and the bot.",
        "When automatic rules can’t pick a category, a short purchase description may be sent to Anthropic (Claude AI) to suggest one. Long numbers, such as card and phone numbers, are removed first, and amounts are not sent. Anthropic does not use this data to train its models.",
        "If voice recognition is enabled, voice notes are sent to a speech-to-text provider for transcription only.",
        "We will list any new provider here before using it.",
      ]],
      ["Where it is stored", [
        "Your data is stored on servers located in the Republic of Uzbekistan. Backups are kept for 14 days and are stored in Uzbekistan too.",
      ]],
      ["How long we keep it", [
        "As long as you have an account. When you delete your account, your data is erased immediately from the live database and disappears from backups within 14 days.",
      ]],
      ["Your rights", [
        "Export: Profile → Data and privacy → Export my data sends all your data to the bot chat as CSV and JSON files.",
        "Correction: you can edit or delete any record in the app or the bot.",
        "Deletion: Profile → Data and privacy → Delete account erases everything.",
        "You can withdraw consent at any time by deleting your account. For other requests contact us through {contact}.",
      ]],
      ["Security", [
        "Every request from the Mini App is verified with Telegram’s signature. Connections are encrypted (HTTPS), access to servers is restricted, and logs never contain amounts or descriptions.",
      ]],
      ["Changes", [
        "If we change this policy, we will show the new version in the app and ask you to accept it again. The date at the top shows the latest version.",
      ]],
    ]),
    terms: doc("Terms of Use", [
      ["The service", [
        "Budget Planner helps you record and plan your own money. It is a tracker: it is not a bank, it does not hold or move money, and it does not give investment, tax or legal advice.",
        "The service is free. Plans, “fits / doesn’t fit” checks and suggestions are estimates based on what you entered. Decisions are yours.",
      ]],
      ["Your account", [
        "Your account is tied to your Telegram account. You are responsible for the accuracy of what you enter and for access to your Telegram.",
        "By using the service you accept these terms and the Privacy Policy.",
      ]],
      ["Fair use", [
        "Don’t try to access other people’s data, overload or break the service, or use it for anything illegal. We may limit or close accounts that do.",
      ]],
      ["Availability", [
        "We work to keep the service running and your data safe, but it is provided “as is”. We may change or stop features, and will warn you in advance when we can.",
        "Keep your own copy of important data: use Export at any time.",
      ]],
      ["Liability", [
        "To the extent the law allows, we are not liable for losses resulting from decisions made using the service or from errors in data you entered.",
      ]],
      ["Changes and law", [
        "We may update these terms. You will see the new version in the app and be asked to accept it.",
        "These terms are governed by the laws of the Republic of Uzbekistan.",
      ]],
    ]),
  },
}
