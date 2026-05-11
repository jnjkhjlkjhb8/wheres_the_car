import 'dart:convert';

import 'package:intl/intl.dart';

List<BusDailyTimeTable> BusDailyTimeTableFromJson(dynamic str) {
  final List<dynamic> jsonData = str is String ? json.decode(str) : str;
  return List<BusDailyTimeTable>.from(jsonData.map((x) => BusDailyTimeTable.fromJson(x)));
}
class StopTime{
  final int StopSequence;
  final String StopUID;
  final String StopID;
  final Map<String,String> StopName;
  final DateTime DepartureTime;
  final int TimeType;
  StopTime({
    required this.StopSequence,
    required this.StopUID,
    required this.DepartureTime,
    required this.StopID,
    required this.StopName,
    required this.TimeType,
  });
  factory StopTime.fromJson(Map<String,dynamic> stopTime){
    return StopTime(
      StopSequence: stopTime["StopSequence"],
      StopUID: stopTime["StopUID"],
      StopID: stopTime["StopID"],
      StopName: Map<String,String>.from(stopTime["StopName"]),
      TimeType: stopTime["TimeType"],
      DepartureTime: DateFormat("HH:mm").parse(stopTime["DepartureTime"]).toLocal(),
    );
  }
}
class Timetable{
  final List<StopTime> StopTimes;
  Timetable({
    required this.StopTimes,
  });
  factory Timetable.fromJson(Map<String,dynamic> timetables){
    return Timetable(
      StopTimes: List<StopTime>.from(timetables["StopTimes"].map((e) => StopTime.fromJson(Map<String, dynamic>.from(e)))),
    );
  }
}
class BusDailyTimeTable {
  final DateTime BusDate;
  final String RouteUID;
  final String SubRouteUID;
  final Map<String, String> SubRouteName;
  final int Direction;
  final List<Timetable> Timetables;
  BusDailyTimeTable({
    required this.BusDate,
    required this.RouteUID,
    required this.SubRouteUID,
    required this.SubRouteName,
    required this.Direction,
    required this.Timetables,
  });
  factory BusDailyTimeTable.fromJson(Map<String, dynamic> busDailyTimeTable){
    return BusDailyTimeTable(
      BusDate: DateTime.parse(busDailyTimeTable["BusDate"]).toLocal(),
      RouteUID: busDailyTimeTable["RouteUID"],
      SubRouteUID: busDailyTimeTable["SubRouteUID"],
      SubRouteName: Map<String, String>.from(busDailyTimeTable["SubRouteName"]),
      Direction: busDailyTimeTable["Direction"],
      Timetables: List<Timetable>.from(busDailyTimeTable["Timetables"].map((e) => Timetable.fromJson(Map<String, dynamic>.from(e)))),
    );
  }
}