import 'dart:convert';

List<DailyTimetable> THSRdailyTimetabletrainFromJson(String str) => List<DailyTimetable>.from(json.decode(str).map((x) => DailyTimetable.fromJson(x)));

class StopTime{
  final int StopSequence;
  final String StationID;
  final Map<String,String> StationName;
  final String ArrivalTime;
  final String DepartureTime;
  StopTime({
    required this.StopSequence,
    required this.StationID,
    required this.StationName,
    required this.ArrivalTime,
    required this.DepartureTime
  });
  factory StopTime.fromJson(Map<String,dynamic> stopTime){
    return StopTime(
      StopSequence: stopTime["StopSequence"],
      StationID: stopTime["StationID"],
      StationName: Map<String,String>.from(stopTime["StationName"]),  
      ArrivalTime: stopTime["ArrivalTime"],
      DepartureTime: stopTime["DepartureTime"]
    );
  }
}
class DailyTimetable{
  final String TrainDate;
  final String TrainNo;
  final int Direction;
  final String StartingStationID;
  final Map<String,String> StartingStationName;
  final String EndingStationID;
  final Map<String,String> EndingStationName;
  final Map<String,String> Note;
  final List<StopTime> StopTimes;
  final DateTime UpdateTime;
  final int VersionID;
  DailyTimetable({
    required this.TrainDate,
    required this.TrainNo,
    required this.Direction,
    required this.StartingStationID,
    required this.StartingStationName,
    required this.EndingStationID,
    required this.EndingStationName,
    required this.Note,
    required this.StopTimes,
    required this.UpdateTime,
    required this.VersionID
  });
  factory DailyTimetable.fromJson(Map<String,dynamic> dailyTimetable){
    return DailyTimetable(
      TrainDate: dailyTimetable["TrainDate"],
      TrainNo: dailyTimetable["TrainNo"],
      Direction: dailyTimetable["Direction"],
      StartingStationID: dailyTimetable["StartingStationID"],
      StartingStationName: Map<String,String>.from(dailyTimetable["StartingStationName"]),
      EndingStationID: dailyTimetable["EndingStationID"],
      EndingStationName: Map<String,String>.from(dailyTimetable["EndingStationName"]),
      Note: Map<String,String>.from(dailyTimetable["Note"]),
      StopTimes: List<StopTime>.from(dailyTimetable["StopTimes"].map((x) => StopTime.fromJson(x))),
      UpdateTime: DateTime.parse(dailyTimetable["UpdateTime"]),
      VersionID: dailyTimetable["VersionID"]
    );
  }
}