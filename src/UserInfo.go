package main

type UserInfo struct {
	username string
	password string
	following []*UserInfo
}