package lib

// COMMANDS (frontend to backend server commands)
const (
	CommandSignup int32 = iota
	CommandDeleteAccount
	CommandLogin
	CommandFollow
	CommandUnfollow
	CommandSearch
	CommandChirp
	CommandGetChirps
)

// STATUS CODES (Status Codes for backend function calls, returned to frontend)
const (
	StatusAccepted = iota
	StatusUserFound
	StatusUserNotFound
	StatusUserFollowed
	StatusUserUnfollowed
	StatusEmpty
	StatusWriteError
	StatusReadError
	StatusEncodeError
	StatusDecodeError
)

var statusText = map[int]string {
	StatusAccepted:       "Command Accepted and Executed Successfully",
	StatusUserFound:      "User Found",
	StatusUserNotFound:   "User Does Not Exist",
	StatusUserFollowed:   "User Followed",
	StatusUserUnfollowed: "User Not Followed",
	StatusEmpty:          "Object is Empty",
	StatusWriteError:     "Binary Write Error",
	StatusReadError:      "Binary Read Error",
	StatusEncodeError:    "Gob Encode Error",
	StatusDecodeError:    "Gob Decode Error",
}

func StatusText(code int) string {
	return statusText[code]
}

type CommandRequest struct {
	CommandCode  int
	Data         interface{}
}

type CommandResponse struct {
	Success  bool
	Status   int
	Data     interface{}
}