import 'dart:convert';

List<MetroNear> MetroNearFromJson(String str) => List<MetroNear>.from(json.decode(str).map((x) => MetroNear.fromJson(x)));

class MetroNear{
  final double? PositionLon;
  final double? PositionLat;
  final String? GeoHash;
  final String? LocationCity;
  final String StationUID;
  final Map<String,String> StationName;
  MetroNear({
    required this.PositionLon,
    required this.PositionLat,
    required this.GeoHash,
    required this.LocationCity,
    required this.StationUID,
    required this.StationName,
  });
  factory MetroNear.fromJson(Map<String, dynamic> json) {
    return MetroNear(
      PositionLon: json['StationPosition']['PositionLon'],
      PositionLat: json['StationPosition']['PositionLat'],
      GeoHash: json['StationPosition']['GeoHash'],
      LocationCity: json['LocationCity'],
      StationUID: json['StationUID'],
      StationName: Map<String, String>.from(json['StationName']),
    );
  }
}