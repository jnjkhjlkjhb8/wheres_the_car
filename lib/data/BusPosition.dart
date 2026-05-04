import 'dart:convert';

List<Busposition> BuspositionFromJson(String str) => List<Busposition>.from(json.decode(str).map((x) => Busposition.fromJson(x)));

class Busposition{
  final String PlateNumb;
  final String SubRouteUID;
  final Map<String,String> SubRouteName;
  final double PositionLon;
  final double PositionLat;
  final String GeoHash;
  final double? Speed;
  final double? Azimuth;
  final int? BusStatus;
  final double GPSTime;
  Busposition(
    {
      required this.SubRouteUID,
      required this.SubRouteName,
      required this.PlateNumb,
      required this.PositionLon,
      required this.PositionLat,
      required this.GeoHash,
      required this.Speed,
      required this.Azimuth,
      required this.BusStatus,
      required this.GPSTime
    }
  );
  factory Busposition.fromJson(Map<String,dynamic> busStationFrequency){
    return Busposition(
      SubRouteUID: busStationFrequency["SubRouteUID"],
      PlateNumb: busStationFrequency["PlateNumb"],
      PositionLon: busStationFrequency['BusPosition']["PositionLon"],
      PositionLat: busStationFrequency['BusPosition']["PositionLat"],
      GeoHash: busStationFrequency['BusPosition']["GeoHash"],
      Speed: busStationFrequency["Speed"],
      Azimuth: busStationFrequency["Azimuth"],
      BusStatus: busStationFrequency["BusStatus"],
      GPSTime: busStationFrequency["GPSTime"],
      SubRouteName: Map<String,String>.from(busStationFrequency["SubRouteName"]),
    );
  }
}