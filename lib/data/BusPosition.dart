import 'dart:convert';

List<Busposition> BusPositionFromJson(String str) => List<Busposition>.from(json.decode(str).map((x) => Busposition.fromJson(x)));

class Busposition {
  final String PlateNumb;
  final String OperatorID;
  final String OperatorNo;
  final String RouteUID;
  final String RouteID;
  final Map<String,String> RouteName;
  final String SubRouteUID;
  final String SubRouteID;
  final Map<String,String> SubRouteName;
  final int Direction;
  final double PositionLon,PositionLat;
  final String GeoHash;
  final double Speed;
  final double Azimuth;
  final int DutyStatus;
  final int BusStatus;
  final int MessageType;
  final DateTime GPSTime;
  final DateTime TransTime;
  final DateTime SrcRecTime;
  final DateTime SrcTransTime;
  final DateTime SrcUpdateTime;
  final DateTime UpdateTime;
  Busposition(
    {
      required this.PlateNumb,
      required this.OperatorID,
      required this.OperatorNo,
      required this.RouteUID,
      required this.RouteID,
      required this.RouteName,
      required this.SubRouteUID,
      required this.SubRouteID,
      required this.SubRouteName,
      required this.Direction,
      required this.PositionLon,
      required this.PositionLat,
      required this.GeoHash,
      required this.Speed,
      required this.Azimuth,
      required this.DutyStatus,
      required this.BusStatus,
      required this.MessageType,
      required this.GPSTime,
      required this.TransTime,
      required this.SrcRecTime,
      required this.SrcTransTime,
      required this.SrcUpdateTime,
      required this.UpdateTime
    }
  );
  factory Busposition.fromJson(Map<String,dynamic> busposition){
    final Stopposition = busposition["StopPosition"] as Map<String, dynamic>;
    return Busposition(
      PlateNumb: busposition["PlateNumb"],
      OperatorID: busposition["OperatorID"],
      RouteUID: busposition["RouteUID"],
      RouteID: busposition["RouteID"],
      RouteName: Map<String,String>.from(busposition["RouteName"]),
      SubRouteUID: busposition["SubRouteUID"],
      SubRouteID: busposition["SubRouteID"],
      SubRouteName: Map<String,String>.from(busposition["SubRouteName"]),
      Direction: busposition["Direction"],
      PositionLon: Stopposition["PositionLon"],
      PositionLat: Stopposition["PositionLat"],
      GeoHash: Stopposition["GeoHash"],
      Speed: busposition["Speed"],
      Azimuth: busposition["Azimuth"],
      DutyStatus: busposition["DutyStatus"],
      BusStatus: busposition["BusStatus"],
      MessageType: busposition["MessageType"],
      GPSTime: DateTime.parse(busposition["GPSTime"]),
      TransTime: DateTime.parse(busposition["TransTime"]),
      SrcRecTime: DateTime.parse(busposition["SrcRecTime"]),
      SrcTransTime: DateTime.parse(busposition["SrcTransTime"]),
      SrcUpdateTime: DateTime.parse(busposition["SrcUpdateTime"]),
      UpdateTime: DateTime.parse(busposition["UpdateTime"]),
      OperatorNo: busposition["OperatorNo"]
    );
  }
}