import 'dart:convert';

List<Busnearbystation> BusnearbystationFromJson(String str) => List<Busnearbystation>.from(json.decode(str).map((x) => Busnearbystation.fromJson(x)));

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
  final Map<String, String> StationName;
  final double PositionLon,PositionLat;
  final String GeoHash;
  final String StationGroupID;
  final String Bearing;
  final DateTime UpdateTime;
  final List<BusStops> stops;
  Busnearbystation(
    {
      required this.StationUID,
      required this.StationName,
      required this.PositionLon,
      required this.PositionLat,
      required this.GeoHash,
      required this.StationGroupID,
      required this.Bearing,
      required this.UpdateTime,
      required this.stops,
    }
  );
  factory Busnearbystation.fromJson(Map<String,dynamic> busnearbystation){
    final stopPosition = busnearbystation["StationPosition"] as Map<String, dynamic>;
    return Busnearbystation(
      StationUID: busnearbystation["StationUID"],
      StationName: Map<String,String>.from(busnearbystation["StationName"]),
      PositionLon: stopPosition["PositionLon"],
      PositionLat: stopPosition["PositionLat"],
      GeoHash: stopPosition["GeoHash"],
      stops: List<BusStops>.from(busnearbystation["Stops"].map((x) => BusStops.fromJson(x))),
      StationGroupID: busnearbystation["StationGroupID"],
      Bearing: busnearbystation["Bearing"],
      UpdateTime: DateTime.parse(busnearbystation["UpdateTime"]),
    );
  }
}