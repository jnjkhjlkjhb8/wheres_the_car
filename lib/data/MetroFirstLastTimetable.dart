import 'dart:convert';

List<FirstLastTimetable> firstLastTimetableFromJson(String str) => List<FirstLastTimetable>.from(json.decode(str).map((x) => FirstLastTimetable.fromJson(x)));

class FirstLastTimetable{
  final DateTime SrcUpdateTime;
  final DateTime UpdateTime;
  final int VersionID;
  final String LineNo;
  final String LineID;
  final String StationID;
  final Map<String, String> StationName;
  final String TripHeadSign;
  final String DestinationStaionID;
  final Map<String, String> DestinationStationName;
  final int TrainType;
  final String FirstTrainTime;
  final String LastTrainTime;
  final String ServiceTag;
  final bool Monday;
  final bool Tuesday;
  final bool Wednesday;
  final bool Thursday;
  final bool Friday;
  final bool Saturday;
  final bool Sunday;
  final bool NationalHolidays;
  FirstLastTimetable(
    {
      required this.SrcUpdateTime,
      required this.UpdateTime,
      required this.VersionID,
      required this.LineNo,
      required this.LineID,
      required this.StationID,
      required this.StationName,
      required this.TripHeadSign,
      required this.DestinationStaionID,
      required this.DestinationStationName,
      required this.TrainType,
      required this.FirstTrainTime,
      required this.LastTrainTime,
      required this.ServiceTag,
      required this.Monday,
      required this.Tuesday,
      required this.Wednesday,
      required this.Thursday,
      required this.Friday,
      required this.Saturday,
      required this.Sunday,
      required this.NationalHolidays
    }
  );
  factory FirstLastTimetable.fromJson(Map<String,dynamic> timetable){
    final S = timetable["ServiceDay"] as Map<String, dynamic>;
    return FirstLastTimetable(
      SrcUpdateTime: DateTime.parse(timetable["SrcUpdateTime"]),
      UpdateTime: DateTime.parse(timetable["UpdateTime"]),
      VersionID: timetable["VersionID"],
      LineNo: timetable["LineNo"],
      LineID: timetable["LineID"],
      StationID: timetable["StationID"],
      StationName: Map<String,String>.from(timetable["StationName"]),
      TripHeadSign: timetable["TripHeadSign"],
      DestinationStaionID: timetable["DestinationStaionID"],
      DestinationStationName: Map<String,String>.from(timetable["DestinationStationName"]),
      TrainType: timetable["TrainType"],
      FirstTrainTime: timetable["FirstTrainTime"],
      LastTrainTime: timetable["LastTrainTime"],
      ServiceTag: timetable["ServiceTag"],
      Monday: S["Monday"],
      Tuesday: S["Tuesday"],
      Wednesday: S["Wednesday"],
      Thursday: S["Thursday"],
      Friday: S["Friday"],
      Saturday: S["Saturday"],
      Sunday: S["Sunday"],
      NationalHolidays: S["NationalHolidays"]
    );
  }
}