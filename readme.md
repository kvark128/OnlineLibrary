# OnlineLibrary

OnlineLibrary — это клиентское Windows-приложение с открытым исходным кодом для онлайн-библиотек предоставляющих доступ по протоколу DAISY Online V1, с поддержкой потокового воспроизведения lkf и mp3-ресурсов.

## Возможности программы

* Добавление любого числа учётных записей различных библиотечных сервисов поддерживающих протокол DAISY Online V1.
* Полная поддержка навигации по библиотечному меню.
* Поиск книг в библиотеке и работа с «книжной полкой».
* Загрузка любых библиотечных ресурсов на локальный диск вашего устройства.
* Потоковое воспроизведение ресурсов форматов lkf и mp3 с регулировкой громкости, высоты и поддержкой перемотки по текущему фрагменту.
* Ускорение воспроизведения книг до трёх раз и замедления до двух раз без изменения высоты звука (используется библиотека sonic).
* Запоминание позиции воспроизведения для последних 256 запущенных книг.
* Работа в портативном режиме с USB-флеш-накопителя.

## Добавление новой учётной записи

Для добавления в программу новой учётной записи, активируйте строку меню → Библиотека → Учётные записи → Добавить учётную запись или нажмите сочетание клавиш Control+N.
Откроется диалоговое окно добавления новой учётной записи со следующими полями:

* Отображаемое имя: Произвольное, человекочитаемое обозначение учётной записи используемое в интерфейсе программы. Например название библиотеки.
* Адрес сервера: URL по которому доступен предоставляемый библиотекой сервер DAISY Online. Например для библиотеки AV3715.ru это https://do.av3715.ru. Префикс https:// в этом поле уже указан.
* Имя пользователя: Имя учётной записи используемое для входа (обычно E-mail). Предоставляется библиотекой при регистрации.
* Пароль: Пароль от учётной записи. Предоставляется библиотекой при регистрации.

После заполнения всех полей и нажатия кнопки OK, OnlineLibrary попытается выполнить вход с указанной учётной записью, и в случае успеха, сохранит её в конфигурационном файле, а пользователю будет показано главное меню библиотеки.
Добавленная таким образом учётная запись станет текущей и будет доступна в строке меню → Библиотека → Учётные записи.
В OnlineLibrary можно добавлять любое количество учётных записей различных библиотек и переключаться между ними через это меню.
Удаление из программы текущей учётной записи выполняется с помощью соответствующего пункта в подменю «Библиотека».
Обратите внимание, что при удалении учётной записи, также удаляются все сохранённые позиции воспроизведения для всех книг запускавшихся из под этой учётной записи.

## Работа с библиотекой

После входа в текущую учётную запись, в окне программы станет доступен список с главным меню библиотеки.
Навигация по этому меню выполняется клавишами-стрелками вверх/вниз, а активация выбранного пункта производится нажатием Enter.
Из строки меню, в подменю «Библиотека» доступны некоторые дополнительные команды навигации по библиотеке, а именно:
* Переход в главное меню библиотеки: Control+M.
* Переход на книжную полку: Control+E.
* Открытие диалога поиска книг: Control+F.
* Переход на предыдущее меню в библиотеке: BackSpace.

При нахождении в списке книг, например на книжной полке или в результатах поиска, доступны следующие операции над выбранной книгой:
* Запуск потокового воспроизведения с последней прослушанной позиции: Enter.
* Загрузка книги на диск: Control+D.
* Добавление книги на книжную полку: Control+A
* Удаление книги с книжной полки: Shift+Delete
* Получение информации о книге (если она предоставляется библиотекой): Control+I.

Для воспроизводимой в данный момент книги доступны следующие операции:
* Воспроизведение / Пауза: Control+K или пробел.
* Остановка воспроизведения с переходом в начало текущего фрагмента: Control+пробел.
* Переход на следующий фрагмент: Control+L.
* Переход на предыдущий фрагмент: Control+J.
* Увеличение громкости: Control+↑.
* Уменьшение громкости: Control+↓.
* Перемотка по фрагменту на 5 секунд вперёд: →.
* Перемотка по фрагменту на 5 секунд назад: ←.
* Перемотка по фрагменту на 30 секунд вперёд: Control+→.
* Перемотка по фрагменту на 30 секунд назад: Control+←.
* Перемотка по фрагменту на 1 минуту вперёд: Shift+→.
* Перемотка по фрагменту на 1 минуту назад: Shift+←.
* Перемотка по фрагменту на 5 минут вперёд: Control+Shift+→.
* Перемотка по фрагменту на 5 минут назад: Control+Shift+←.
* Ускорение воспроизведения: Control+C.
* Замедление воспроизведения: Control+X.
* Сброс скорости воспроизведения к значению по умолчанию: Control+Z.
* Увеличение высоты звука: Shift+C.
* Уменьшение высоты звука: Shift+X.
* Сброс высоты звука к значению по умолчанию: Shift+Z.
* Переход к первому фрагменту книги: Control+BackSpace.
* Переход к указанному фрагменту книги: Control+G.
* Переход к указанной позиции в текущем фрагменте: Shift+G.

Прошедшее и общее время текущего фрагмента, а также его номер и общее число фрагментов в книге, отображается во время воспроизведения в строке состояния.

## Рабочий каталог OnlineLibrary

При запуске программы создаётся каталог %USERPROFILE%\OnlineLibrary, который используется для хранения загружаемых книг.
Также там располагается файл конфигурации (config.json) и журнал последней сессии работы программы (session.log).
При желании, рядом с исполняемым файлом программы можно создать пустую папку OnlineLibrary.
В этом случае, именно эта папка будет использоваться в качестве рабочего каталога для хранения книг, конфигурации и журнала работы, делая программу полностью портативной.

## Пожертвование
Если вам понравилась данная программа и вы хотите повысить мотивацию автора к её дальнейшему развитию, то это можно сделать переводом любой суммы на следующий кошелёк YooMoney (бывшие Яндекс.Деньги):
https://yoomoney.ru/to/410012293543375
