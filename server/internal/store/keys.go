package store

import "fmt"

func BusEtaRoute(subRouteUID string) string       { return "bus_eta_route:" + subRouteUID }
func BusEtaStation(city, name string) string      { return fmt.Sprintf("bus_eta_station:%s:%s", city, name) }
func BusDailyTimetable(subRouteUID string) string { return "bus_daily_timetable:" + subRouteUID }
func BusFare(subRouteUID string) string           { return "bus_fare:" + subRouteUID }

func BikeAvailability(stationUID string) string { return "bike_availability:" + stationUID }

func MrtLive(system, stationID, lineID string) string {
	return fmt.Sprintf("mrt_live:%s:%s:%s", system, stationID, lineID)
}
func MrtLiveChannel(system, stationID string) string {
	return fmt.Sprintf("mrt_live:%s:%s", system, stationID)
}

const TraDelayAll = "tra:delay:all"
const TraDelayHash = "tra:delay"

func TraDelayTrain(trainNo string) string  { return "tra:delay:" + trainNo }
func TraLiveboard(stationID string) string { return "tra:liveboard:" + stationID }

func TraFare(origin, dest string) string { return fmt.Sprintf("TRA_Fare:%s:%s", origin, dest) }
func TraTimetable(date, origin, dest string) string {
	return fmt.Sprintf("TRA_timetable:%s:%s:%s", date, origin, dest)
}
func TraStoptimes(date, trainNo string) string {
	return fmt.Sprintf("TRA_Stoptimes:%s:%s", date, trainNo)
}

func ThsrFare(origin, dest string) string { return fmt.Sprintf("THSR_Fare:%s:%s", origin, dest) }
func ThsrTimetable(date, origin, dest string) string {
	return fmt.Sprintf("THSR_timetable:%s:%s:%s", date, origin, dest)
}
func ThsrStoptimes(date, trainNo string) string {
	return fmt.Sprintf("THSR_Stoptimes:%s:%s", date, trainNo)
}
func ThsrSeats(date, trainNo string) string { return fmt.Sprintf("thsr_seats:%s:%s", date, trainNo) }

func MaasPlan(hash string) string { return "maas:plan:" + hash }

func TDXLastModified(name string) string { return "LastTimeGet_" + name }

func MqttKey(topic string) string { return "mqtt:" + replaceSep(topic) }

func replaceSep(s string) string {
	out := make([]byte, len(s))
	for i := range s {
		if s[i] == '/' {
			out[i] = ':'
		} else {
			out[i] = s[i]
		}
	}
	return string(out)
}
