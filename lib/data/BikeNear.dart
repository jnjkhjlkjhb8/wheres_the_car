import 'dart:convert';

List<Bikenear> BikenearFromJson(dynamic data) {
  if (data is String) {
    return List<Bikenear>.from(json.decode(data).map((x) => Bikenear.fromJson(x)));
  } else if (data is List) return List<Bikenear>.from(data.map((x) => Bikenear.fromJson(x)));
  return [];
}
List<Bikeaavailability> BikeaavailabilityFromJson(dynamic data) {
  if (data is String) {
    return List<Bikeaavailability>.from(json.decode(data).map((x) => Bikeaavailability.fromJson(x)));
  } else if (data is List) return List<Bikeaavailability>.from(data.map((x) => Bikeaavailability.fromJson(x)));
  return [];
}

class BikeStation{
  final String StationUID;
  final Map<String,String> StationName;
  final double PositionLon;
  final double PositionLat;
  final String GeoHash;
  final int BikesCapacity;
  final int ServiceType;
  final int ServiceStatus;
  final int AvailableReturnBikes;
  final Map<String,int> AvailableRentBikesDetail;
  final DateTime UpdateTime;
  BikeStation({
    required this.StationUID,
    required this.StationName,
    required this.PositionLon,
    required this.PositionLat,
    required this.GeoHash,
    required this.BikesCapacity,
    required this.ServiceType,
    required this.ServiceStatus,
    required this.AvailableReturnBikes,
    required this.AvailableRentBikesDetail,
    required this.UpdateTime,
  });
  factory BikeStation.fromJson(Map<String, dynamic> json) {
    return BikeStation(
      StationUID: json['StationUID'],
      StationName: Map<String,String>.from(json['StationName']),
      PositionLon: json['StationPosition']['PositionLon'].toDouble(),
      PositionLat: json['StationPosition']['PositionLat'].toDouble(),
      GeoHash: json['StationPosition']['GeoHash'],
      BikesCapacity: json['BikesCapacity'],
      ServiceType: json['ServiceType'],
      ServiceStatus: json['ServiceStatus'],
      AvailableReturnBikes: json['AvailableReturnBikes'],
      AvailableRentBikesDetail: Map<String, int>.from(json['AvailableRentBikesDetail']),
      UpdateTime: DateTime.parse(json['UpdateTime']),
    );
  }
}

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
  final Map<String,int> AvailableRentBikesDetail;
  final DateTime SrcUpdateTime;
  Bikeaavailability({
    required this.StationUID,
    required this.ServiceStatus,
    required this.AvailableReturnBikes,
    required this.AvailableRentBikesDetail,
    required this.SrcUpdateTime,
  });
  factory Bikeaavailability.fromJson(Map<String, dynamic> json) {
    return Bikeaavailability(
      StationUID: json['StationUID'],
      ServiceStatus: json['ServiceStatus'],
      AvailableReturnBikes: json['AvailableReturnBikes'],
      AvailableRentBikesDetail: Map<String, int>.from(json['AvailableRentBikesDetail']),
      SrcUpdateTime: DateTime.parse(json['SrcUpdateTime']),
    );
  }
}