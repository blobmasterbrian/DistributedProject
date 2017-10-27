package lib

// COMMANDS (frontend to backend server commands)
const (
	CommandSignup = iota
	CommandDeleteAccount
	CommandLogin
	CommandFollow
	CommandUnfollow
	CommandSearch
	CommandChirp
	CommandGetChirps
)

// STATUS CODES (Status Codes for frontend/backend communication)
const (
	StatusAccepted = iota
	StatusUserFound
	StatusUserNotFound
	StatusUserFollowed
	StatusUserNotFollowed
    StatusIncorrectPassword
    StatusDuplicateUser
    StatusConnectionError
	StatusInternalError
	StatusEncodeError
	StatusDecodeError
)

var statusText = map[int]string {
	StatusAccepted:          "Command Accepted and Executed Successfully",
	StatusUserFound:         "User Found",
	StatusUserNotFound:      "User Does Not Exist",
	StatusUserFollowed:      "User Followed",
	StatusUserNotFollowed:   "User Not Followed",
    StatusIncorrectPassword: "Password Is Incorrect",
    StatusDuplicateUser:     "User Already Exists",
    StatusConnectionError:   "Server Connection Error",
	StatusInternalError:     "I'm sorry dave, I'm afriad I can't do that",
	StatusEncodeError:       "Gob Encode Error",
	StatusDecodeError:       "Gob Decode Error",
}

func StatusText(code int) string {
	return statusText[code]
}

type CommandRequest struct {
	CommandCode int
	Data        interface{}
}

type CommandResponse struct {
	Success bool
	Status  int
	Data    interface{}
}
