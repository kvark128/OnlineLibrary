package msg

type Message struct {
	Code MessageCode
	Data interface{}
}

type MessageCode uint32

const (
	ACTIVATE_MENU MessageCode = iota
	OPEN_BOOKSHELF
	OPEN_NEWBOOKS
	MAIN_MENU
	DOWNLOAD_BOOK
	BOOK_DESCRIPTION
	ISSUE_BOOK
	REMOVE_BOOK
	SEARCH_BOOK
	MENU_BACK
	SET_PROVIDER
	LIBRARY_ADD
	LIBRARY_REMOVE
	LIBRARY_INFO
	PLAYER_SPEED_RESET
	PLAYER_SPEED_UP
	PLAYER_SPEED_DOWN
	PLAYER_VOLUME_RESET
	PLAYER_VOLUME_UP
	PLAYER_VOLUME_DOWN
	PLAYER_PLAY_PAUSE
	PLAYER_STOP
	PLAYER_OFFSET_FRAGMENT
	PLAYER_OFFSET_POSITION
	PLAYER_GOTO_FRAGMENT
	PLAYER_GOTO_POSITION
	PLAYER_OUTPUT_DEVICE
	PLAYER_SET_TIMER
	BOOKMARK_SET
	BOOKMARK_FETCH
	LOG_SET_LEVEL
)
