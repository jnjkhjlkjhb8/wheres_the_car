import 'dart:convert';

List<LiveBoard> liveBoardFromJson(String str) => List<LiveBoard>.from(json.decode(str).map((x) => LiveBoard.fromJson(x)));

class LiveBoard{
  final String LineNO;
  final String LineID;
  final Map<String, String> LineName;
  final String StationID;
  final Map<String, String> StationName;
  final String TripHeadSign;
  final String DestinationStaionID;
  final String DestinationStationID;
  final Map<String, String> DestinationStationName;
  final int ServiceStatus;
  final int EstimateTime;
  final DateTime SrcUpdateTime;
  final DateTime UpdateTime;
  LiveBoard({
    required this.LineNO,
    required this.LineID,
    required this.LineName,
    required this.StationID,
    required this.StationName,
    required this.TripHeadSign,
    required this.DestinationStaionID,
    required this.DestinationStationID,
    required this.DestinationStationName,
    required this.ServiceStatus,
    required this.EstimateTime,
    required this.SrcUpdateTime,
    required this.UpdateTime
  });
  factory LiveBoard.fromJson(Map<String, dynamic> json) {
    return LiveBoard(
      LineNO: json['LineNO'] as String,
      LineID: json['LineID'] as String,
      LineName: Map<String, String>.from(json['LineName']),
      StationID: json['StationID'] as String,
      StationName: Map<String, String>.from(json['StationName']),
      TripHeadSign: json['TripHeadSign'] as String,
      DestinationStaionID: json['DestinationStaionID'] as String,
      DestinationStationID: json['DestinationStationID'] as String,
      DestinationStationName: Map<String, String>.from(json['DestinationStationName']),
      ServiceStatus: json['ServiceStatus'] as int,
      EstimateTime: json['EstimateTime'] as int,
      SrcUpdateTime: DateTime.parse(json['SrcUpdateTime'] as String),
      UpdateTime: DateTime.parse(json['UpdateTime'] as String),
    );
  }
}