import 'dart:convert';

List<MetroNear> MetroNearFromJson(dynamic data) {
  if (data is String) {
    return List<MetroNear>.from(json.decode(data).map((x) => MetroNear.fromJson(x)));
  } else if (data is List) return List<MetroNear>.from(data.map((x) => MetroNear.fromJson(x)));
  return [];
}

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
      PositionLon: json['StationPosition']['PositionLon'].toDouble(),
      PositionLat: json['StationPosition']['PositionLat'].toDouble(),
      GeoHash: json['StationPosition']['GeoHash'],
      LocationCity: json['LocationCity'],
      StationUID: json['StationUID'],
      StationName: Map<String, String>.from(json['StationName']),
    );
  }
}