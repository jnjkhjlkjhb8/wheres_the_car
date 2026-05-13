import 'dart:convert';

List<THSRDailytime> thsrDailytimeFromJson(String str) => List<THSRDailytime>.from(json.decode(str).map((x) => THSRDailytime.fromJson(x)));

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
class THSRDailytime{
  final String TrainDate;
  final String TrainNo;
  final int Direction;
  final String StartingStationID;
  final Map<String,String> StartingStationName;
  final String EndingStationID;
  final Map<String,String> EndingStationName;
  final Map<String,String> Note;
  final StopTime OriginStopTime;
  final StopTime DestinationStopTime;
  final DateTime UpdateTime;
  final int VersionID;
  THSRDailytime({
    required this.TrainDate,
    required this.TrainNo,
    required this.Direction,
    required this.StartingStationID,
    required this.StartingStationName,
    required this.EndingStationID,
    required this.EndingStationName,
    required this.Note,
    required this.OriginStopTime,
    required this.DestinationStopTime,
    required this.UpdateTime,
    required this.VersionID
  });
  factory THSRDailytime.fromJson(Map<String,dynamic> dailyTimetable){
    return THSRDailytime(
      TrainDate: dailyTimetable["TrainDate"],
      TrainNo: dailyTimetable["TrainNo"],
      Direction: dailyTimetable["Direction"],
      StartingStationID: dailyTimetable["StartingStationID"],
      StartingStationName: Map<String,String>.from(dailyTimetable["StartingStationName"]),
      EndingStationID: dailyTimetable["EndingStationID"],
      EndingStationName: Map<String,String>.from(dailyTimetable["EndingStationName"]),
      Note: Map<String,String>.from(dailyTimetable["Note"]),
      OriginStopTime: StopTime.fromJson(dailyTimetable["OriginStopTime"]),
      DestinationStopTime: StopTime.fromJson(dailyTimetable["DestinationStopTime"]),
      UpdateTime: DateTime.parse(dailyTimetable["UpdateTime"]),
      VersionID: dailyTimetable["VersionID"]
    );
  }
}