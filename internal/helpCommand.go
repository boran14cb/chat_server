package internal

import (
	"github.com/fatih/color"
)

// Colours for text
var (
	cyan   = color.New(color.FgCyan).SprintFunc()
	green  = color.New(color.FgGreen).SprintFunc()
	blue   = color.New(color.FgBlue).SprintFunc()
	purple = color.New(color.FgMagenta).SprintFunc()
	red    = color.New(color.FgRed).SprintFunc()
	yellow = color.New(color.FgYellow).SprintFunc()
)

// Help messages
var (
	USAGE     [3]string = [3]string{red("\nUsage:"), yellow(" /<Command>"), cyan(" arguments")}
	NAME      [3]string = [3]string{red("/name"), yellow(" <new_name>"), cyan(" (Sets new username)")}
	MSG       [3]string = [3]string{red("/msg"), yellow(" <receiver_username> <message>"), cyan(" (Sends a DM)")}
	BROADCAST [3]string = [3]string{red("/all"), yellow(" <message>"), cyan(" (Sends a message to all users in the current room")}
	SPAM      [3]string = [3]string{red("/spam"), yellow(" <spam_n_times> <message>"), cyan(" (Spams the room 'N' times)")}
	SHOUT     [3]string = [3]string{red("/shout"), yellow(" <message>"), cyan(" (Sends a message to room in capitals")}
	CREATE    [3]string = [3]string{red("/create"), yellow(" <room_name>"), cyan(" (creates a new room with the specified name)")}
	JOIN      [3]string = [3]string{red("/join"), yellow(" <room_name>"), cyan(" (Joins a room)")}
	KICK      [3]string = [3]string{red("/kick"), yellow(" <username>"), cyan(" (Kicks the user out of the room, you have to be admin)")}
	PROMOTE   [3]string = [3]string{red("/promote"), yellow(" <username>"), cyan(" (promotes a user to a mod in the room)")}
	ROOMS     [3]string = [3]string{red("/rooms"), yellow(""), cyan(" (shows the available rooms)")}
	QUIT      [3]string = [3]string{red("/quit"), yellow(""), cyan(" (Quits the room)")}
	EXIT      [3]string = [3]string{red("/exit"), yellow(""), cyan(" (Close the client connection)")}
	HELP      [3]string = [3]string{red("/help"), yellow(""), cyan(" (Lists all commands)")}
	LIST      [3]string = [3]string{red("/list"), yellow(" <(optional) room_name>"), cyan(" (Lists active users)\n")}
)
