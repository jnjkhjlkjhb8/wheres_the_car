import 'dart:convert';

List<ODFare> TRAODFareFromJson(String str) => List<ODFare>.from(json.decode(str).map((x) => ODFare.fromJson(x)));

class Fare{
  final String TicketType;
  final double Price;
  Fare({
    required this.TicketType,
    required this.Price
  });
  factory Fare.fromJson(Map<String, dynamic> fare){
    return Fare(
      TicketType: fare["TicketType"] as String,
      Price: fare["Price"] as double
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
  final int VersionID;
  ODFare({
    required this.OriginStationID,
    required this.OriginStationName,
    required this.DestinationStationID,
    required this.DestinationStationName,
    required this.Direction,
    required this.Fares,
    required this.SrcUpdateTime,
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
      VersionID: odfare["VersionID"] as int
    );
  }
}