import 'dart:convert';

List<DailyTimetableTrain> TRAdailyTimetabletrainFromJson(String str) => List<DailyTimetableTrain>.from(json.decode(str).map((x) => DailyTimetableTrain.fromJson(x)));

class StopTime{
  final int StopSequence;
  final String StopID;
  final Map<String, String> StationName;
  final String ArrivalTime;
  final String DepartureTime;
  final int SuspendedFlag;
  StopTime({
    required this.StopSequence,
    required this.StopID,
    required this.StationName,
    required this.ArrivalTime,
    required this.DepartureTime,
    required this.SuspendedFlag,
  });
  factory StopTime.fromJson(Map<String, dynamic> json) => StopTime(
    StopSequence: json["StopSequence"],
    StopID: json["StopID"],
    StationName: Map.from(json["StationName"]).map((k, v) => MapEntry<String, String>(k, v)),
    ArrivalTime: json["ArrivalTime"],
    DepartureTime: json["DepartureTime"],
    SuspendedFlag: json["SuspendedFlag"],
  );
}
class DailyTimetableTrain{
  final String TrainDate;
  final String TrainNo;
  final int Direction;
  final String StartingStationID;
  final List<StopTime> StartingStationName;
  final String EndingStationID;
  final List<StopTime> EndingStationName;
  final String TripHeadsign;
  final String TrainTypeID;
  final String TrainTypeCode;
  final Map<String, String> TrainTypeName;
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
  final List<StopTime> StopTimes;
  final DateTime UpdateTime;
  final int VersionID;
  DailyTimetableTrain({
    required this.TrainNo,
    required this.Direction,
    required this.StopTimes,
    required this.TrainDate,
    required this.StartingStationID,
    required this.StartingStationName,
    required this.EndingStationID,
    required this.EndingStationName,
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
    required this.SuspendedFlag,
    required this.Note,
    required this.UpdateTime,
    required this.VersionID,
  });
  factory DailyTimetableTrain.fromJson(Map<String, dynamic> json){
    final DailyTrainInfo = json["DailyTrainInfo"] as Map<String, dynamic>;
    return DailyTimetableTrain(
      TrainNo: DailyTrainInfo["TrainNo"],
      Direction: DailyTrainInfo["Direction"],
      StopTimes: List<StopTime>.from(DailyTrainInfo["StopTimes"].map((x) => StopTime.fromJson(x))),
      TrainDate: json["TrainDate"],
      StartingStationID: DailyTrainInfo["StartingStationID"],
      StartingStationName: List<StopTime>.from(DailyTrainInfo["StartingStationName"].map((x) => StopTime.fromJson(x))),
      EndingStationID: DailyTrainInfo["EndingStationID"],
      EndingStationName: List<StopTime>.from(DailyTrainInfo["EndingStationName"].map((x) => StopTime.fromJson(x))),
      TripHeadsign: DailyTrainInfo["TripHeadsign"],
      TrainTypeID: DailyTrainInfo["TrainTypeID"],
      TrainTypeCode: DailyTrainInfo["TrainTypeCode"],
      TrainTypeName: Map.from(DailyTrainInfo["TrainTypeName"]).map((zh, en) => MapEntry<String, String>(zh, en)),
      TripLine: DailyTrainInfo["TripLine"],
      OverNightStationID: DailyTrainInfo["OverNightStationID"],
      WheelchairFlag: DailyTrainInfo["WheelchairFlag"],
      PackageServiceFlag: DailyTrainInfo["PackageServiceFlag"],
      DiningFlag: DailyTrainInfo["DiningFlag"],
      BikeFlag: DailyTrainInfo["BikeFlag"],
      BreastFeedingFlag: DailyTrainInfo["BreastFeedingFlag"],
      DailyFlag: DailyTrainInfo["DailyFlag"],
      ServiceAddedFlag: DailyTrainInfo["ServiceAddedFlag"],
      SuspendedFlag: DailyTrainInfo["SuspendedFlag"],
      Note: Map.from(DailyTrainInfo["Note"]).map((zh, en) => MapEntry<String, String>(zh, en)),
      UpdateTime: DateTime.parse(json["UpdateTime"]),
      VersionID: json["VersionID"],
    );
  }
}