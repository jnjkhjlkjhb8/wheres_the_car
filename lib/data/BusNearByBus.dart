import 'dart:convert';

List<BusNearByBus> BusNearByBusFromJson(String str) => List<BusNearByBus>.from(json.decode(str).map((x) => BusNearByBus.fromJson(x)));

class BusNearByBus{
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
  BusNearByBus(
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
  factory BusNearByBus.fromJson(Map<String,dynamic> bus){
    final stopPosition = bus["StopPosition"] as Map<String, dynamic>;
    return BusNearByBus(
      PlateNumb: bus["PlateNumb"],
      OperatorID: bus["OperatorID"],
      OperatorNo: bus["OperatorNo"],
      RouteUID: bus["RouteUID"],
      RouteID: bus["RouteID"],
      RouteName: Map<String,String>.from(bus["RouteName"]),
      SubRouteUID: bus["SubRouteUID"],
      SubRouteID: bus["SubRouteID"],
      SubRouteName: Map<String,String>.from(bus["SubRouteName"]),
      Direction: bus["Direction"],
      PositionLon: stopPosition["PositionLon"],
      PositionLat: stopPosition["PositionLat"],
      GeoHash: stopPosition["GeoHash"],
      Speed: bus["Speed"],
      Azimuth: bus["Azimuth"],
      DutyStatus: bus["DutyStatus"],
      BusStatus: bus["BusStatus"],
      MessageType: bus["MessageType"],
      GPSTime: DateTime.parse(bus["GPSTime"]),
      TransTime: DateTime.parse(bus["TransTime"]),
      SrcRecTime: DateTime.parse(bus["SrcRecTime"]),
      SrcTransTime: DateTime.parse(bus["SrcTransTime"]),
      SrcUpdateTime: DateTime.parse(bus["SrcUpdateTime"]),
      UpdateTime: DateTime.parse(bus["UpdateTime"])
    );
  }
}