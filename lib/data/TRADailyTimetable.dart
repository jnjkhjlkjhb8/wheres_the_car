import 'dart:convert';

List<TRADailytime> traDailytimeFromJson(String str) => List<TRADailytime>.from(json.decode(str).map((x) => TRADailytime.fromJson(x)));

class StopTime{
  final int StopSequence;
  final String StationID;
  final Map<String,String> StationName;
  final String ArrivalTime;
  final String DepartureTime;
  final int SuspendedFlag;
  StopTime({
    required this.StopSequence,
    required this.StationID,
    required this.StationName,
    required this.ArrivalTime,
    required this.DepartureTime,
    required this.SuspendedFlag
  });
  factory StopTime.fromJson(Map<String,dynamic> stopTime){
    return StopTime(
      StopSequence: stopTime["StopSequence"],
      StationID: stopTime["StationID"],
      StationName: Map<String,String>.from(stopTime["StationName"]),  
      ArrivalTime: stopTime["ArrivalTime"],
      DepartureTime: stopTime["DepartureTime"],
      SuspendedFlag: stopTime["SuspendedFlag"]
    );
  }
}
class TRADailytime{
  final String TrainDate;
  final String TrainNo;
  final int Direction;
  final String StartingStationID;
  final Map<String,String> StartingStationName;
  final String EndingStationID;
  final Map<String,String> EndingStationName;
  final String TripHeadsign;
  final String TrainTypeID;
  final String TrainTypeCode;
  final Map<String,String> TrainTypeName;
  final int TripLine;
  final String OverNightStationID;
  final int WheelchairFlag;
  final int PackageServiceFlag;
  final int DiningFlag;
  final int BikeFlag;
  final int BreastFeedingFlag;
  final int DailyFlag;
  final int ServiceAddedFlag;
  final int SuspendedFlag;
  final Map<String, String> Note;
  final StopTime DestinationStopTime;
  final StopTime OriginStopTime;
  final DateTime UpdateTime;
  final int VersionID;
  TRADailytime({
    required this.TrainDate,
    required this.TrainNo,
    required this.Direction,
    required this.StartingStationID,
    required this.StartingStationName,
    required this.EndingStationID,
    required this.EndingStationName,
    required this.Note,
    required this.DestinationStopTime,
    required this.OriginStopTime,
    required this.UpdateTime,
    required this.VersionID,
    required this.TripHeadsign,
    required this.TrainTypeID,
    required this.TrainTypeCode,
    required this.TrainTypeName,
    required this.TripLine,
    required this.OverNightStationID,
    required this.WheelchairFlag,
    required this.PackageServiceFlag,
    required this.DiningFlag,
    required this.BikeFlag,
    required this.BreastFeedingFlag,
    required this.DailyFlag,
    required this.ServiceAddedFlag, 
    required this.SuspendedFlag
  });
  factory TRADailytime.fromJson(Map<String,dynamic> dailyTimetable){
    final DailyTrainInfo = dailyTimetable["DailyTrainInfo"] as Map<String, dynamic>;
    return TRADailytime(
      TrainDate: dailyTimetable["TrainDate"],
      TrainNo: DailyTrainInfo["TrainNo"],
      Direction: DailyTrainInfo["Direction"],
      StartingStationID: DailyTrainInfo["StartingStationID"],
      StartingStationName: Map<String,String>.from(DailyTrainInfo["StartingStationName"]),
      EndingStationID: DailyTrainInfo["EndingStationID"],
      EndingStationName: Map<String,String>.from(DailyTrainInfo["EndingStationName"]),
      Note: Map<String,String>.from(DailyTrainInfo["Note"]),
      DestinationStopTime: StopTime.fromJson(dailyTimetable["DestinationStopTime"]),
      OriginStopTime: StopTime.fromJson(dailyTimetable["OriginStopTime"]),
      UpdateTime: DateTime.parse(dailyTimetable["UpdateTime"]),
      VersionID: dailyTimetable["VersionID"],
      TripHeadsign: DailyTrainInfo["TripHeadsign"],
      TrainTypeID: DailyTrainInfo["TrainTypeID"],
      TrainTypeCode: DailyTrainInfo["TrainTypeCode"],
      TrainTypeName: Map<String,String>.from(DailyTrainInfo["TrainTypeName"]),
      TripLine: DailyTrainInfo["TripLine"],
      OverNightStationID: DailyTrainInfo["OverNightStationID"],
      WheelchairFlag: DailyTrainInfo["WheelchairFlag"],
      PackageServiceFlag: DailyTrainInfo["PackageServiceFlag"],
      DiningFlag: DailyTrainInfo["DiningFlag"],
      BikeFlag: DailyTrainInfo["BikeFlag"],
      BreastFeedingFlag: DailyTrainInfo["BreastFeedingFlag"],
      DailyFlag: DailyTrainInfo["DailyFlag"],
      ServiceAddedFlag: DailyTrainInfo["ServiceAddedFlag"],
      SuspendedFlag: DailyTrainInfo["SuspendedFlag"]
    );
  }
}