import 'dart:convert';

List<Busnearbystation> BusnearbystationFromJson(dynamic str) {
  final List<dynamic> jsonData = str is String ? json.decode(str) : str;
  return List<Busnearbystation>.from(jsonData.map((x) => Busnearbystation.fromJson(x)));
}
List<BusStops> BusStopsFromJson(dynamic str) {
  final List<dynamic> jsonData = str is String ? json.decode(str) : str;
  return List<BusStops>.from(jsonData.map((x) => BusStops.fromJson(x)));
}

class BusStops{
  final String StopUID;
  final Map<String,String> StopName;
  final String RouteUID;
  final Map<String,String> RouteName;
  BusStops(
    {
      required this.StopUID,
      required this.StopName,
      required this.RouteUID,
      required this.RouteName
    }
  );
  factory BusStops.fromJson(Map<String,dynamic> stop){
    return BusStops(
      StopUID: stop["StopUID"],
      StopName: Map<String,String>.from(stop["StopName"]),
      RouteUID: stop["RouteUID"],
      RouteName: Map<String,String>.from(stop["RouteName"])
    );
  }
}
class Busnearbystation {
  final String StationUID;
  final String StationID;
  final Map<String, String> StationName;
  final double PositionLon,PositionLat;
  final String GeoHash;
  final String? StationGroupID;
  final String Bearing;
  final DateTime UpdateTime;
  final List<BusStops> stops;
  final String LocationCityCode;
  Busnearbystation(
    {
      required this.StationUID,
      required this.StationID,
      required this.StationName,
      required this.PositionLon,
      required this.PositionLat,
      required this.GeoHash,
      required this.StationGroupID,
      required this.Bearing,
      required this.UpdateTime,
      required this.stops,
      required this.LocationCityCode
    }
  );
  factory Busnearbystation.fromJson(Map<String,dynamic> busnearbystation){
    return Busnearbystation(
      StationUID: busnearbystation["StationUID"]?.toString() ?? '',
      StationID: busnearbystation["StationID"]?.toString() ?? '',
      StationName: Map<String,String>.from(busnearbystation["StationName"]),
      PositionLon: busnearbystation["StationPosition"]["PositionLon"].toDouble(),
      PositionLat: busnearbystation["StationPosition"]["PositionLat"].toDouble(),
      GeoHash: busnearbystation["StationPosition"]["GeoHash"].toString(),
      stops: BusStopsFromJson(busnearbystation["Stops"]),
      StationGroupID: busnearbystation["StationGroupID"].toString(),
      Bearing: busnearbystation["Bearing"].toString(),
      LocationCityCode: busnearbystation["LocationCityCode"].toString(),
      UpdateTime: DateTime.parse(busnearbystation["UpdateTime"],
      )
    );
  }
}