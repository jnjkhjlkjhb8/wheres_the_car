import 'dart:convert';

List<ODFare> THSRODFareFromJson(String str) => List<ODFare>.from(json.decode(str).map((x) => ODFare.fromJson(x)));

class Fare{
  final int TicketType;
  final int FareClass;
  final int CabinClass;
  final int Price;
  Fare({
    required this.TicketType,
    required this.FareClass,
    required this.CabinClass,
    required this.Price
  });
  factory Fare.fromJson(Map<String, dynamic> fare){
    return Fare(
      TicketType: fare["TicketType"] as int,
      FareClass: fare["FareClass"] as int,
      CabinClass: fare["CabinClass"] as int,
      Price: fare["Price"] as int
    );  
  }
}
class ODFare{
  final String OriginStationID;
  final Map<String, String> OriginStationName;
  final String DestinationStationID;
  final Map<String, String> DestinationStationName;
  final int Direction;
  final List<Fare> Fares;
  final DateTime SrcUpdateTime;
  final DateTime UpdateTime;
  final int VersionID;
  ODFare({
    required this.OriginStationID,
    required this.OriginStationName,
    required this.DestinationStationID,
    required this.DestinationStationName,
    required this.Direction,
    required this.Fares,
    required this.SrcUpdateTime,
    required this.UpdateTime,
    required this.VersionID
  });
  factory ODFare.fromJson(Map<String, dynamic> odfare){
    return ODFare(
      OriginStationID: odfare["OriginStationID"] as String,
      OriginStationName: Map<String, String>.from(odfare["OriginStationName"]),
      DestinationStationID: odfare["DestinationStationID"] as String,
      DestinationStationName: Map<String, String>.from(odfare["DestinationStationName"]),
      Direction: odfare["Direction"] as int,
      Fares: List<Fare>.from(odfare["Fares"].map((x) => Fare.fromJson(x))),
      SrcUpdateTime: DateTime.parse(odfare["Src UpdateTime"] as String),
      UpdateTime: DateTime.parse(odfare["UpdateTime"] as String),
      VersionID: odfare["VersionID"] as int
    );
  }
}