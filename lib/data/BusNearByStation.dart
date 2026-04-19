import 'dart:convert';

List<Busnearbystation> BusnearbystationFromJson(String str) => List<Busnearbystation>.from(json.decode(str).map((x) => Busnearbystation.fromJson(x)));

class BusStops{
  final String StopUID;
  final String StopID;
  final Map<String,String> StopName;
  final String RouteUID;
  final String RouteID;
  final Map<String,String> RouteName;
  BusStops(
    {
      required this.StopUID,
      required this.StopID,
      required this.StopName,
      required this.RouteUID,
      required this.RouteID,
      required this.RouteName
    }
  );
  factory BusStops.fromJson(Map<String,dynamic> stop){
    return BusStops(
      StopUID: stop["StopUID"],
      StopID: stop["StopID"],
      StopName: Map<String,String>.from(stop["StopName"]),
      RouteUID: stop["RouteUID"],
      RouteID: stop["RouteID"],
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
  final String StationAddress;
  final String StationGroupID;
  final String LocationCityCode;
  final String Bearing;
  final DateTime UpdateTime;
  final int VersionID;
  Busnearbystation(
    {
      required this.StationUID,
      required this.StationID,
      required this.StationName,
      required this.PositionLon,
      required this.PositionLat,
      required this.GeoHash,
      required this.StationAddress,
      required this.StationGroupID,
      required this.LocationCityCode,
      required this.Bearing,
      required this.UpdateTime,
      required this.VersionID
    }
  );
  factory Busnearbystation.fromJson(Map<String,dynamic> busnearbystation){
    final stopPosition = busnearbystation["StopPosition"] as Map<String, dynamic>;
    return Busnearbystation(
      StationUID: busnearbystation["StationUID"],
      StationID: busnearbystation["StationID"],
      StationName: Map<String,String>.from(busnearbystation["StationName"]),
      PositionLon: stopPosition["PositionLon"],
      PositionLat: stopPosition["PositionLat"],
      GeoHash: stopPosition["GeoHash"],
      StationAddress: busnearbystation["StationAddress"],
      StationGroupID: busnearbystation["StationGroupID"],
      LocationCityCode: busnearbystation["LocationCityCode"],
      Bearing: busnearbystation["Bearing"],
      UpdateTime: DateTime.parse(busnearbystation["UpdateTime"]),
      VersionID: busnearbystation["VersionID"]
    );
  }
}