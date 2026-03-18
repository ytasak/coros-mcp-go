package main

import "os"

var baseURL = getBaseURL()

func getBaseURL() string {
	if v := os.Getenv("COROS_BASE_URL"); v != "" {
		return v
	}
	return "https://teamapi.coros.com"
}

// workoutTypes maps (mode, subMode) to human-readable sport names.
var workoutTypes = map[[2]int]string{
	{8, 1}: "Outdoor Run", {8, 2}: "Indoor Run",
	{9, 1}: "Outdoor Bike", {9, 2}: "Indoor Bike", {9, 3}: "E-Bike",
	{9, 4}: "Mountain Bike", {9, 5}: "E-Mountain Bike", {9, 6}: "Gravel Bike",
	{10, 1}: "Open Water", {10, 2}: "Pool Swim",
	{13, 1}: "Triathlon", {13, 2}: "Multisport",
	{14, 1}: "Mountain Climb", {15, 1}: "Trail Run", {16, 1}: "Hike",
	{18, 1}: "GPS Cardio", {18, 2}: "Gym Cardio",
	{19, 1}: "XC Ski", {20, 1}: "Track Run",
	{21, 1}: "Ski", {21, 2}: "Snowboard",
	{23, 2}: "Strength", {24, 1}: "Rowing", {24, 2}: "Indoor Rower",
	{31, 1}: "Walk", {34, 2}: "Jump Rope",
	{98, 1}: "Custom Outdoor", {99, 2}: "Custom Indoor",
}

// authErrorCodes indicates token expiry or auth failure.
var authErrorCodes = map[string]bool{
	"1003": true,
	"1004": true,
	"1005": true,
	"1019": true,
	"5006": true,
}
