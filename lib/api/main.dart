import 'dart:convert';
import 'package:bus/data/BusNear.dart';
import 'package:bus/data/BusStationEstimateTime.dart';
import 'package:bus/data/MetroFirstLastTimetable.dart';
import 'package:bus/data/MetroLiveBoard.dart';
import 'package:bus/data/MetroPrice.dart';
import 'package:bus/data/THSRDailyTimetableTrain.dart';
import 'package:bus/data/THSRDailyTimetable.dart';
import 'package:bus/data/THSRODFare.dart';
import 'package:bus/data/TRADailyTimetable.dart';
import 'package:bus/data/TRADailyTimetableTrain.dart';
import 'package:bus/data/TRAODFare.dart';
import 'package:dio/dio.dart';
import 'package:pretty_dio_logger/pretty_dio_logger.dart';
import '../data/BusStopOfRoute.dart';
import '../utility/database.dart';
import '../data/BusEstimateTime.dart';
import '../data/BusPosition.dart';
import '../data/BusShape.dart';
import '../data/BusRoute.dart';
import '../data/MetroNear.dart';
import '../data/BikeNear.dart';

class Tdx{
  Dio _dio = Dio()..interceptors.add(PrettyDioLogger());
  Database _db = Database();
  late String _accesstoken;
  static const String _CilentID = String.fromEnvironment("TDX_CLIENT_ID"); // API 位置
  static const String _CilentSecret = String.fromEnvironment("TDX_CLIENT_SECRET"); // API 位置
  Tdx(){
    _accesstoken = _db.getData("token") ?? "";
  }
  final Map<String, String> _cites = {
    "TPE":"Taipei",
    "NWT": "NewTaipei",
    "TAO": "Taoyuan",
    "TXG": "Taichung",
    "TNN": "Tainan",
    "KHH": "Kaohsiung",
    "KEE": "Keelung",
    "HSZ": "Hsinchu",
    "HSQ": "HsinchuCounty",
    "MIA": "MiaoliCounty",
    "CHA": "ChanghuaCounty",
    "NAN": "NantouCounty",
    "YUN": "YunlinCounty",
    "CYQ": "ChiayiCounty",
    "CYI": "Chiayi",
    "PIF": "PingtungCounty",
    "ILA": "YilanCounty",
    "HUA": "HualienCounty",
    "TTT": "TaitungCounty",
    "KIN": "KinmenCounty",
    "PEN": "PenghuCounty",
    "LIE": "LienchiangCounty",
  }; // size = 19
  Future<String> getToken() async{
    try{
      Response response = await _dio.post(
        "https://tdx.transportdata.tw/auth/realms/TDXConnect/protocol/openid-connect/token",
        data: {
          "grant_type": "client_credentials",
          "client_id": _CilentID,
          "client_secret": _CilentSecret,
        },
        options: Options(
          headers: {
            "Content-Type": "application/x-www-form-urlencoded",
          },
        ),
      );
      if(response.statusCode == 200){
        _accesstoken = response.data["access_token"];
        if (Database().getData("token") == null) Database().saveData("token", _accesstoken);
        else Database().updateData("token", _accesstoken);
        print("get token: $_accesstoken");
        return _accesstoken;
      }else{
        throw Exception("Failed to get token");
      }
    }
    on DioException catch (e){
      throw Exception("Failed to get token: $e");
    }
  }
  Future<List<BusStopOfRoute>> getBusStopOfRoute(String city,String route) async{
    try{
      Response response = await _dio.get(
        "https://tdx.transportdata.tw/api/basic/v2/Bus/StopOfRoute/City/$city",
        queryParameters: {
          '\$select': "RouteUID,RouteName,SubRouteUID,SubRouteName,Direction,Stops,UpdateTime",
          '\$filter': "RouteUID eq '$route'",
          '\$format': 'JSON',
        },
        options: Options(
          headers: {
            "authorization": "Bearer $_accesstoken",
            "Content-Encoding": "br,gzip" ,
            "Accept": "application/json",
          },
        )
      );
      if(response.statusCode == 200){
        return busStopOfRouteFromJson(response.data);
      }else{
        throw Exception("Failed to get bus route");
      }
    }
    on DioException catch (e){
      if(e.response?.statusCode == 401){
        await getToken();
        return getBusStopOfRoute(city, route);
      }
      rethrow;
    }
  }
  Future<List<BusEstimates>> getBusEstimatedTimeOfArrival(String city,String route) async{
    try{
      Response response = await _dio.get(
        "https://tdx.transportdata.tw/api/basic/v2/Bus/EstimatedTimeOfArrival/City/$city",
        queryParameters: {
          '\$select': "PlateNumb,StopUID,StopID,StopName,RouteUID,RouteID,RouteName,SubRouteUID,SubRouteID,SubRouteName,Direction,EstimateTime,ScheduledTime,CurrentStop,DestinationStop,StopSequence,StopStatus,MessageType,NextBusTime,IsLastBus,Estimates,UpdateTime",
          '\$filter': "RouteUID eq '$route'",
          '\$format': 'JSON',
        },
        options: Options(
          headers: {
            "authorization": "Bearer $_accesstoken",
            "Content-Encoding": "br,gzip" ,
            "Accept": "application/json",
          },
        )
      );
      if(response.statusCode == 200){
        return busEstimatesFromJson(response.data);
      }else{
        throw Exception("Failed to get bus route");
      }
    }
    on DioException catch (e){
      if(e.response?.statusCode == 401){
        await getToken();
        return getBusEstimatedTimeOfArrival(city, route);
      }
      rethrow;
    }
  }
  Future<Object> getBusPosition(String city,String route) async{
    try{
      Response response = await _dio.get(
        "https://tdx.transportdata.tw/api/basic/v2/Bus/RealTimeByFrequency/City/$city/$route?\$format=JSON",
        options: Options(
          headers: { "authorization": "Bearer $_accesstoken","Content-Encoding": "br,gzip" },
        )
      );
      if(response.statusCode == 200){
        return BuspositionFromJson(response.data);
      }else{
        throw Exception("Failed to get bus route");
      }
    }
    on DioException catch (e){
      if(e.response?.statusCode == 401){
        await getToken();
        return getBusPosition(city, route);
      }
      rethrow;
    }
  }
  Future<Object> getBusRoute(String city,String route) async{
    try{
      Response response = await _dio.get(
        "https://tdx.transportdata.tw/api/basic/v2/Bus/Route/City/$city/$route?\$format=JSON",
        options: Options(
          headers: { "authorization": "Bearer $_accesstoken","Content-Encoding": "br,gzip" },
        )
      );
      if(response.statusCode == 200){
        return BusRouteFromJson(response.data);
      }else{
        throw Exception("Failed to get bus route");
      }
    }
    on DioException catch (e){
      if(e.response?.statusCode == 401){
        await getToken();
        return getBusRoute(city, route);
      }
      rethrow;
    }
  }
  Future<Object> getBusShape(String route) async{
    try{
      Response response = await _dio.get(
        "https://tdx.transportdata.tw/api/basic/v2/Bus/Shape/City/",
          queryParameters: {
            '\$select': "SubRouteUID,SubRouteName,Direction,EncodedPolyline,UpdateTime",
            '\$filter': "SubRouteUID eq '$route'",
            '\$format': 'JSON',
          },
        options: Options(
          headers: {
            "authorization": "Bearer $_accesstoken",
            "Content-Encoding": "br,gzip"
          },
        )
      );
      if(response.statusCode == 200){
        return BusShapeFromJson(response.data);
      }else{
        throw Exception("Failed to get bus route");
      }
    }
    on DioException catch (e){
      if(e.response?.statusCode == 401){
        await getToken();
        return getBusShape(route);
      }
      rethrow;
    }
  }
  Future<List<Busnearbystation>> getBusNearByStation(double Lon,double Lat,int range) async{
    try{
      Response response = await _dio.get(
        "https://tdx.transportdata.tw/api/advanced/v2/Bus/Station/NearBy",
          queryParameters: {
            '\$select': "StationUID,StationID,StationName,StationPosition,StationGroupID,Bearing,LocationCityCode,UpdateTime,Stops",
            '\$spatialFilter': "nearby(${Lat},${Lon},${range})",
            '\$format': 'JSON',
          },
        options: Options(
          headers: {
            "authorization": "Bearer $_accesstoken",
            "Content-Encoding": "br,gzip"
          },
        )
      );
      if(response.statusCode == 200){
        return BusnearbystationFromJson(response.data);
      }else{
        throw Exception("Failed to get bus route");
      }
    }
    on DioException catch (e){
      if(e.response?.statusCode == 401){
        await getToken();
        return getBusNearByStation(Lon, Lat,range);
      }
      rethrow;
    }
  }
  Future<List<MetroNear>> getMetroNear(double Lon,double Lat,int range) async{
    try{
      Response response = await _dio.get(
          "https://tdx.transportdata.tw/api/advanced/v2/Rail/Metro/Station/NearBy",
          queryParameters: {
            '\$select': "StationPosition,LocationCity,StationUID,StationName",
            '\$spatialFilter': "nearby(${Lat},${Lon},${range})",
            '\$format': 'JSON',
          },
          options: Options(
            headers: {
              "authorization": "Bearer $_accesstoken",
              "Content-Encoding": "br,gzip"
            },
          )
      );
      if(response.statusCode == 200){
        return MetroNearFromJson(response.data);
      }else{
        throw Exception("Failed to get bus route");
      }
    }
    on DioException catch (e){
      if(e.response?.statusCode == 401){
        await getToken();
        return getMetroNear(Lon, Lat,range);
      }
      rethrow;
    }
  }
  Future<List<BikeStation>> getBikeNear(double Lon,double Lat,int range) async{
    try{
      final response = await Future.wait([
        _dio.get(
          "https://tdx.transportdata.tw/api/advanced/v2/Bike/Station/NearBy",
          queryParameters: {
            '\$select': "StationUID,StationName,StationPosition,BikesCapacity,ServiceType",
            '\$spatialFilter': "nearby(${Lat},${Lon},${range})",
            '\$format': 'JSON',
          },
          options: Options(
            headers: {
              "authorization": "Bearer $_accesstoken",
              "Content-Encoding": "br,gzip"
            },
          )
        ),
        _dio.get(
          "https://tdx.transportdata.tw/api/advanced/v2/Bike/Availability/NearBy",
          queryParameters: {
            '\$select': "StationUID,ServiceStatus,AvailableReturnBikes,AvailableRentBikesDetail,SrcUpdateTime",
            '\$spatialFilter': "nearby(${Lat},${Lon},${range})",
            '\$format': 'JSON',
          },
          options: Options(
            headers: {
              "authorization": "Bearer $_accesstoken",
              "Content-Encoding": "br,gzip"
            },
          )
        ),
      ]);
      List<dynamic> stationData = response[0].data is String ? json.decode(response[0].data) : response[0].data;
      List<dynamic> availabilityData = response[1].data is String ? json.decode(response[1].data) : response[1].data;
      final mp = {
        for (var i in availabilityData) i['StationUID'] : i
      };
      return stationData.map((e) {
        final availability = mp[e['StationUID']];
        return BikeStation(
          StationUID: e['StationUID'] ?? "",
          StationName: Map<String, String>.from(e['StationName'] ?? {}),
          PositionLon: (e['StationPosition']?['PositionLon'] ?? 0.0).toDouble(),
          PositionLat: (e['StationPosition']?['PositionLat'] ?? 0.0).toDouble(),
          GeoHash: e['StationPosition']?['GeoHash'] ?? "",
          BikesCapacity: e['BikesCapacity'] ?? 0,
          ServiceType: e['ServiceType'] ?? 0,
          ServiceStatus: availability?['ServiceStatus'] ?? 0,
          AvailableReturnBikes: availability?['AvailableReturnBikes'] ?? 0,
          AvailableRentBikesDetail: Map<String, int>.from(availability?['AvailableRentBikesDetail'] ?? {}),
          UpdateTime: availability != null && availability['SrcUpdateTime'] != null
              ? DateTime.parse(availability['SrcUpdateTime'])
              : DateTime.now(),
        );
      }).toList();
    }
    on DioException catch (e){
      if(e.response?.statusCode == 401){
        await getToken();
        return getBikeNear(Lon, Lat,range);
      }
      rethrow;
    }
  }
  Future<Object> getBusPositionByStation(String City,String StationUID) async{
    try{
      Response response = await _dio.get(
        "https://tdx.transportdata.tw/api/advanced/v2/Bus/RealTimeByFrequency/City/${_cites[City]!}/PassThrough/Station/${StationUID}",
        queryParameters: {
          '\$select': "PlateNumb,SubRouteUID,SubRouteName,BusPosition,Speed,Azimuth,BusStatus,GPSTime",
          '\$format': 'JSON',
        },
        options: Options(
          headers: {
            "authorization": "Bearer $_accesstoken",
            "Content-Encoding": "br,gzip"
          },
        )
      );
      if(response.statusCode == 200){
        return BuspositionFromJson(response.data);
      }else{
        throw Exception("Failed to get bus route");
      }
    }
    on DioException catch (e){
      if(e.response?.statusCode == 401){
        await getToken();
        return getBusPositionByStation(City, StationUID);
      }
      rethrow;
    }
  }
  Future<List<BusN1EstimateTime>> getBusEstimateByStation(String City,String StationUID) async {
    String? city = _cites[City];
    if (city == null) {
      print("City code not found: $City");
    }
    try{
      Response response = await _dio.get(
          "https://tdx.transportdata.tw/api/advanced/v2/Bus/EstimatedTimeOfArrival/City/${city}/PassThrough/Station/${StationUID}",
          queryParameters: {
            '\$select': "PlateNumb,SubRouteUID,SubRouteName,EstimateTime,ScheduledTime,DestinationStop,StopStatus,NextBusTime,IsLastBus,Estimates,UpdateTime",
            '\$format': 'JSON',
          },
          options: Options(
            headers: {
              "authorization": "Bearer $_accesstoken",
              "Content-Encoding": "br,gzip"
            },
          )
      );
      if(response.statusCode == 200){
        return busN1EstimateTimeFromJson(response.data);
      }else{
        throw Exception("Failed to get bus route");
      }
    }
    on DioException catch (e){
      if(e.response?.statusCode == 401){
        await getToken();
        return getBusEstimateByStation(City, StationUID);
      }
      rethrow;
    }
  }
  Future<List<BusN1EstimateTime>> getInterBusEstimateByStation(String StationUID) async{
    try{
      Response response = await _dio.get(
          "https://tdx.transportdata.tw/api/advanced/v2/Bus/EstimatedTimeOfArrival/InterCity/PassThrough/Station/${StationUID}",
          queryParameters: {
            '\$select': "PlateNumb,SubRouteUID,SubRouteName,EstimateTime,ScheduledTime,DestinationStop,StopStatus,NextBusTime,IsLastBus,Estimates,UpdateTime",
            '\$format': 'JSON',
          },
          options: Options(
            headers: {
              "authorization": "Bearer $_accesstoken",
              "Content-Encoding": "br,gzip"
            },
          )
      );
      if(response.statusCode == 200){
        return getInterBusEstimateByStation(response.data);
      }else{
        throw Exception("Failed to get bus route");
      }
    }
    on DioException catch (e){
      if(e.response?.statusCode == 401){
        await getToken();
        return getInterBusEstimateByStation(StationUID);
      }
      rethrow;
    }
  }
  Future<List<BusStopOfRoute>> getInterBusStopOfRoute(String route,String route2) async{
    try{
      Response response = await _dio.get(
        "https://tdx.transportdata.tw/api/basic/v2/Bus/StopOfRoute/InterCity",
        queryParameters: {
          '\$select': "RouteUID,RouteName,SubRouteUID,SubRouteName,Direction,Stops,UpdateTime",
          '\$filter': "SubRouteUID eq '$route' or SubRouteUID eq '$route2'",
          '\$format': 'JSON',
        },
        options: Options(
          headers: {
            "authorization": "Bearer $_accesstoken",
            "Content-Encoding": "br,gzip" ,
            "Accept": "application/json",
          },
        )
      );
      if(response.statusCode == 200){
        return busStopOfRouteFromJson(response.data);
      }else{
        throw Exception("Failed to get bus route");
      }
    }
    on DioException catch (e){
      if(e.response?.statusCode == 401){
        await getToken();
        return getInterBusStopOfRoute(route,route2);
      }
      rethrow;
    }
  }
  Future<List<BusEstimates>> getInterBusEstimatedTimeOfArrival(String route,String route2) async{
    try{
      Response response = await _dio.get(
        "https://tdx.transportdata.tw/api/basic/v2/Bus/EstimatedTimeOfArrival/InterCity",
          queryParameters: {
            '\$select': "PlateNumb,StopUID,Direction,EstimateTime,ScheduledTime,IsLastBus,Estimates,UpdateTime",
            '\$filter': "SubRouteUID eq '$route' or SubRouteUID eq '$route2'",
            '\$format': 'JSON',
          },
        options: Options(
          headers: {
            "authorization": "Bearer $_accesstoken",
            "Content-Encoding": "br,gzip",
            "Accept": "application/json",
          },
        )
      );
      if(response.statusCode == 200){
        return busEstimatesFromJson(response.data);
      }else{
        throw Exception("Failed to get bus route");
      }
    }
    on DioException catch (e){
      if(e.response?.statusCode == 401){
        await getToken();
        return getInterBusEstimatedTimeOfArrival(route,route2);
      }
      rethrow;
    }
  }
  Future<Object> getInterBusPosition(String city,String route) async{
    try{
      Response response = await _dio.get(
        "https://tdx.transportdata.tw/api/basic/v2/Bus/RealTimeByFrequency/Streaming/InterCity/$city/$route?\$format=JSON",
        options: Options(
          headers: { "authorization": "Bearer $_accesstoken","Content-Encoding": "br,gzip" },
        )
      );
      if(response.statusCode == 200){
        return BuspositionFromJson(response.data);
      }else{
        throw Exception("Failed to get bus route");
      }
    }
    on DioException catch (e){
      if(e.response?.statusCode == 401){
        await getToken();
        return getInterBusPosition(city, route);
      }
      rethrow;
    }
  }
  Future<Object> getInterBusRoute(String city,String route) async{
    try{
      Response response = await _dio.get(
        "https://tdx.transportdata.tw/api/basic/v2/Bus/Route/InterCity/$city/$route?\$format=JSON",
        options: Options(
          headers: { "authorization": "Bearer $_accesstoken","Content-Encoding": "br,gzip" },
        )
      );
      if(response.statusCode == 200){
        return BusRouteFromJson(response.data);
      }else{
        throw Exception("Failed to get bus route");
      }
    }
    on DioException catch (e){
      if(e.response?.statusCode == 401){
        await getToken();
        return getInterBusRoute(city, route);
      }
      rethrow;
    }
  }
  Future<Object> getInterBusShape(String city,String route) async{
    try{
      Response response = await _dio.get(
        "https://tdx.transportdata.tw/api/basic/v2/Bus/Shape/InterCity/$city/$route?\$format=JSON",
        options: Options(
          headers: { "authorization": "Bearer $_accesstoken","Content-Encoding": "br,gzip" },
        )
      );
      if(response.statusCode == 200){
        return BusShapeFromJson(response.data);
      }else{
        throw Exception("Failed to get bus route");
      }
    }
    on DioException catch (e){
      if(e.response?.statusCode == 401){
        await getToken();
        return getInterBusShape(city, route);
      }
      rethrow;
    }
  }
  Future<Object> getMetroFirstLastTimetable(String RailSystem,String Name) async{
    try{
      Response response = await _dio.get(
        "https://tdx.transportdata.tw/api/basic/v2/Rail/Metro/FirstLastTimetable/$RailSystem",
        options: Options(
          headers: { "authorization": "Bearer $_accesstoken","Content-Encoding": "br,gzip" },
        ),
        queryParameters: {
        '\$format': 'JSON',
        '\$filter': "StationName/Zh_tw eq '$Name'", 
        },
      );
      if(response.statusCode == 200){
        return firstLastTimetableFromJson(response.data);
      }else{
        throw Exception("Failed to get bus route");
      }
    }
    on DioException catch (e){
      if(e.response?.statusCode == 401){
        await getToken();
        return getMetroFirstLastTimetable(RailSystem, Name);
      }
      rethrow;
    }
  }
  Future<Object> getMetroPrice(String RailSystem,String Start,String End) async{
    try{
      Response response = await _dio.get(
        "https://tdx.transportdata.tw/api/basic/v2/Rail/Metro/ODFare/$RailSystem",
        options: Options(
          headers: { "authorization": "Bearer $_accesstoken","Content-Encoding": "br,gzip" },
        ),
        queryParameters: {
        '\$format': 'JSON',
        '\$filter': "OriginStationName/Zh_tw eq '$Start' and DestinationStationName/Zh_tw eq '$End'", 
        },
      );
      if(response.statusCode == 200){
        return MRTODFareFromJson(response.data);
      }else{
        throw Exception("Failed to get bus route");
      }
    }
    on DioException catch (e){
      if(e.response?.statusCode == 401){
        await getToken();
        return getMetroPrice(RailSystem, Start, End);
      }
      rethrow;
    }
  }
  Future<Object> getMetroLiveBoard(String RailSystem,String StationName) async{
    try{
      Response response = await _dio.get(
        "https://tdx.transportdata.tw/api/basic/v2/Rail/Metro/LiveBoard/$RailSystem",
        options: Options(
          headers: { "authorization": "Bearer $_accesstoken","Content-Encoding": "br,gzip" },
        ),
        queryParameters: {
        '\$format': 'JSON',
        '\$filter': "StationName/Zh_tw eq '$StationName'", 
        },
      );
      if(response.statusCode == 200){
        return liveBoardFromJson(response.data);
      }else{
        throw Exception("Failed to get bus route");
      }
    }
    on DioException catch (e){
      if(e.response?.statusCode == 401){
        await getToken();
        return getMetroLiveBoard(RailSystem, StationName);
      }
      rethrow;
    }
  }
  Future<Object> getTHSRDailyTimetableTrain(String TrainNo,String TrainDate) async{
    try{
      Response response = await _dio.get(
        "https://tdx.transportdata.tw/api/basic/v2/Rail/THSR/DailyTimetable/TrainNo/$TrainNo/TrainDate/$TrainDate",
        options: Options(
          headers: { "authorization": "Bearer $_accesstoken","Content-Encoding": "br,gzip" },
        ),
        queryParameters: {
        '\$format': 'JSON',
        },
      );
      if(response.statusCode == 200){
        return THSRdailyTimetabletrainFromJson(response.data);
      }else{
        throw Exception("Failed to get bus route");
      }
    }
    on DioException catch (e){
      if(e.response?.statusCode == 401){
        await getToken();
        return getTHSRDailyTimetableTrain(TrainNo, TrainDate);
      }
      rethrow;
    }
  }
  Future<Object> getTHSRDailyTimetable(String OriginStationID,String DestinationStationID,String TrainDate) async{
    try{
      Response response = await _dio.get(
        "https://tdx.transportdata.tw/api/basic/v2/Rail/THSR/DailyTimetable/OD/$OriginStationID/to/$DestinationStationID/$TrainDate",
        options: Options(
          headers: { "authorization": "Bearer $_accesstoken","Content-Encoding": "br,gzip" },
        ),
        queryParameters: {
        '\$format': 'JSON',
        },
      );
      if(response.statusCode == 200){
        return thsrDailytimeFromJson(response.data);
      }else{
        throw Exception("Failed to get bus route");
      }
    }
    on DioException catch (e){
      if(e.response?.statusCode == 401){
        await getToken();
        return getTHSRDailyTimetable(OriginStationID, DestinationStationID, TrainDate);
      }
      rethrow;
    }
  }
  Future<Object> getTHSRODFare(String OriginStationID,String DestinationStationID) async{
    try{
      Response response = await _dio.get(
        "https://tdx.transportdata.tw/api/basic/v2/Rail/THSR/ODFare/$OriginStationID/to/$DestinationStationID",
        options: Options(
          headers: { "authorization": "Bearer $_accesstoken","Content-Encoding": "br,gzip" },
        ),
        queryParameters: {
        '\$format': 'JSON',
        },
      );
      if(response.statusCode == 200){
        return THSRODFareFromJson(response.data);
      }else{
        throw Exception("Failed to get bus route");
      }
    }
    on DioException catch (e){
      if(e.response?.statusCode == 401){
        await getToken();
        return getTHSRODFare(OriginStationID, DestinationStationID);
      }
      rethrow;
    }
  }
  Future<Object> getTRAODFare(String OriginStationID,String DestinationStationID) async{
    try{
      Response response = await _dio.get(
        "https://tdx.transportdata.tw/api/basic/v2/Rail/TRA/ODFare/$OriginStationID/to/$DestinationStationID",
        options: Options(
          headers: { "authorization": "Bearer $_accesstoken","Content-Encoding": "br,gzip" },
        ),
        queryParameters: {
        '\$format': 'JSON',
        },
      );
      if(response.statusCode == 200){
        return TRAODFareFromJson(response.data);
      }else{
        throw Exception("Failed to get bus route");
      }
    }
    on DioException catch (e){
      if(e.response?.statusCode == 401){
        await getToken();
        return getTRAODFare(OriginStationID, DestinationStationID);
      }
      rethrow;
    }
  }
  Future<Object> getTRADailyTimetableTrain(String TrainNo,String TrainDate) async{
    try{
      Response response = await _dio.get(
        "https://tdx.transportdata.tw/api/basic/v2/Rail/TRA/DailyTimetable/TrainNo/$TrainNo/TrainDate/$TrainDate",
        options: Options(
          headers: { "authorization": "Bearer $_accesstoken","Content-Encoding": "br,gzip" },
        ),
        queryParameters: {
        '\$format': 'JSON',
        },
      );
      if(response.statusCode == 200){
        return TRAdailyTimetabletrainFromJson(response.data);
      }else{
        throw Exception("Failed to get bus route");
      }
    }
    on DioException catch (e){
      if(e.response?.statusCode == 401){
        await getToken();
        return getTRADailyTimetableTrain(TrainNo, TrainDate);
      }
      rethrow;
    }
  }
  Future<Object> getTRADailyTimetable(String OriginStationID,String DestinationStationID,String TrainDate) async{
    try{
      Response response = await _dio.get(
        "https://tdx.transportdata.tw/api/basic/v2/Rail/TRA/DailyTimetable/OD/$OriginStationID/to/$DestinationStationID/$TrainDate",
        options: Options(
          headers: { "authorization": "Bearer $_accesstoken","Content-Encoding": "br,gzip" },
        ),
        queryParameters: {
        '\$format': 'JSON',
        },
      );
      if(response.statusCode == 200){
        return traDailytimeFromJson(response.data);
      }else{
        throw Exception("Failed to get bus route");
      }
    }
    on DioException catch (e){
      if(e.response?.statusCode == 401){
        await getToken();
        return getTRADailyTimetable(OriginStationID, DestinationStationID, TrainDate);
      }
      rethrow;
    }
  }
  Future<Object> getBusStationEstimateTime(String StationID,String City) async{
    try{
      Response response = await _dio.get(
        "https://tdx.transportdata.tw/api/advanced/v2/Bus/EstimatedTimeOfArrival/City/$City/PassThrough/Station/$StationID",
        options: Options(
          headers: { "authorization": "Bearer $_accesstoken","Content-Encoding": "br,gzip" },
        ),
        queryParameters: {
        '\$format': 'JSON',
        },
      );
      if(response.statusCode == 200){
        return busN1EstimateTimeFromJson(response.data);
      }else{
        throw Exception("Failed to get bus route");
      }
    }
    on DioException catch (e){
      if(e.response?.statusCode == 401){
        await getToken();
        return getBusStationEstimateTime(StationID, City);
      }
      rethrow;
    }
  }
  Future<Object> getInterBusStationEstimateTime(String StationID) async{
    try{
      Response response = await _dio.get(
        "https://tdx.transportdata.tw/api/advanced/v2/Bus/EstimatedTimeOfArrival/InterCity/PassThrough/Station/$StationID",
        options: Options(
          headers: { "authorization": "Bearer $_accesstoken","Content-Encoding": "br,gzip" },
        ),
        queryParameters: {
        '\$format': 'JSON',
        },
      );
      if(response.statusCode == 200){
        return busN1EstimateTimeFromJson(response.data);
      }else{
        throw Exception("Failed to get bus route");
      }
    }
    on DioException catch (e){
      if(e.response?.statusCode == 401){
        await getToken();
        return getInterBusStationEstimateTime(StationID);
      }
      rethrow;
    }
  }
}