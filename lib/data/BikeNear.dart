import 'dart:convert';



class Bikenear {
  final String StationUID;
  final Map<String,String> StationName;
  final double PositionLon;
  final double PositionLat;
  final String GeoHash;
  final int BikesCapacity;
  final int ServiceType;
  Bikenear({
    required this.StationUID,
    required this.StationName,
    required this.PositionLon,
    required this.PositionLat,
    required this.GeoHash,
    required this.BikesCapacity,
    required this.ServiceType,
  });
  factory Bikenear.fromJson(Map<String, dynamic> json) {
    return Bikenear(
      StationUID: json['StationUID'],
      StationName: Map<String,String>.from(json['StationName']),
      PositionLon: json['StationPosition']['PositionLon'],
      PositionLat: json['StationPosition']['PositionLat'],
      GeoHash: json['StationPosition']['GeoHash'],
      BikesCapacity: json['BikesCapacity'],
      ServiceType: json['ServiceType'],
    );
  }
}
class Bikeaavailability{
  final String StationUID;
  final int ServiceStatus;
  final int AvailableReturnBikes;
  final Map<int,int> AvailableRentBikesDetail;
  final DateTime UpdateTime;
  Bikeaavailability({
    required this.StationUID,
    required this.ServiceStatus,
    required this.AvailableReturnBikes,
    required this.AvailableRentBikesDetail,
    required this.UpdateTime,
  });
  factory Bikeaavailability.fromJson(Map<String, dynamic> json) {
    return Bikeaavailability(
      StationUID: json['StationUID'],
      ServiceStatus: json['ServiceStatus'],
      AvailableReturnBikes: json['AvailableReturnBikes'],
      AvailableRentBikesDetail: Map<int, int>.from(json['AvailableRentBikesDetail']),
      UpdateTime: DateTime.parse(json['UpdateTime']),
    );
  }
}