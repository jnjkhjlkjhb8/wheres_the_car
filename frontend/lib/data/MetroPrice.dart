import 'dart:convert';

List<ODFare> MRTODFareFromJson(String str) => List<ODFare>.from(json.decode(str).map((x) => ODFare.fromJson(x)));

class Fare{
  final int TicketType;
  final int FareClass;
  final String SaleType;
  final String CitizenCode;
  final int Price;
  Fare({
    required this.TicketType,
    required this.FareClass,
    required this.SaleType,
    required this.CitizenCode,
    required this.Price
  });
  factory Fare.fromJson(Map<String,dynamic> fare){
    return Fare(
      TicketType: fare["TicketType"],
      FareClass: fare["FareClass"],
      SaleType: fare["SaleType"],
      CitizenCode: fare["CitizenCode"],
      Price: fare["Price"]
    );
  }
}
class ODFare{
  final DateTime SrcUpdateTime;
  final DateTime UpdateTime;
  final int VersionID;
  final String OriginStationID;
  final Map<String,String> OriginStationName;
  final String DestinationStationID;
  final Map<String,String> DestinationStationName;
  final int TrainType;
  final List<Fare> Fares;
  final int TravelTime;
  final int TravelDistance;
  ODFare({
    required this.SrcUpdateTime,
    required this.UpdateTime,
    required this.VersionID,
    required this.OriginStationID,
    required this.OriginStationName,
    required this.DestinationStationID,
    required this.DestinationStationName,
    required this.TrainType,
    required this.Fares,
    required this.TravelTime,
    required this.TravelDistance
  });
  factory ODFare.fromJson(Map<String,dynamic> odFare){
    return ODFare(
      SrcUpdateTime: DateTime.parse(odFare["SrcUpdateTime"]),
      UpdateTime: DateTime.parse(odFare["UpdateTime"]),
      VersionID: odFare["VersionID"],
      OriginStationID: odFare["OriginStationID"],
      OriginStationName: Map<String,String>.from(odFare["OriginStationName"]),
      DestinationStationID: odFare["DestinationStationID"],
      DestinationStationName: Map<String,String>.from(odFare["DestinationStationName"]),
      TrainType: odFare["TrainType"],
      Fares: odFare["Fares"],
      TravelTime: odFare["TravelTime"],
      TravelDistance: odFare["TravelDistance"]
    );
  }
}