import 'package:bus/data/BusNearByBus.dart';
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

class Tdx{
  Dio _dio = Dio()..interceptors.add(PrettyDioLogger());
  Database _db = Database();
  late String _accesstoken;
  static const String _CilentID = String.fromEnvironment("TDX_CLIENT_ID"); // API 位置
  static const String _CilentSecret = String.fromEnvironment("TDX_CLIENT_SECRET"); // API 位置
  Tdx(){
    _accesstoken = _db.getData("token") ?? "";
  }
  final List<(String,String)> _cites = [
    ("Taipei", "台北市"),
    ("NewTaipei", "新北市"),
    ("Taoyuan", "桃園市"),
    ("Taichung", "台中市"),
    ("Tainan", "台南市"),
    ("Kaohsiung", "高雄市"),
    ("Keelung", "基隆市"),
    ("Hsinchu", "新竹市"),
    ("HsinchuCounty", "新竹縣"),
    ("MiaoliCounty", "苗栗縣"),
    ("ChanghuaCounty", "彰化縣"),
    ("NantouCounty", "南投縣"),
    ("YunlinCounty", "雲林縣"),
    ("ChiayiCounty", "嘉義縣"),
    ("Chiayi", "嘉義市"),
    ("PingtungCounty", "屏東縣"), 
    ("YilanCounty", "宜蘭縣"),
    ("HualienCounty", "花蓮縣"),
    ("TaitungCounty", "台東縣"),
    ("KinmenCounty", "金門縣"),
    ("PenghuCounty", "澎湖縣"),
    ("LienchiangCounty", "連江縣")
  ]; // size = 19
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
        return BusPositionFromJson(response.data);
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
  Future<Object> getBusNearByStation(double Lon,double Lat,int range) async{
    try{
      Response response = await _dio.get(
        "https://tdx.transportdata.tw/api/advanced/v2/Bus/Station/NearBy",
          queryParameters: {
            '\$select': "StationUID,StationName,StopPosition,StationGroupID,Bearing,UpdateTime,Stops",
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
  Future<Object> getMetroNear(double Lon,double Lat,int range) async{
    try{
      Response response = await _dio.get(
          "https://tdx.transportdata.tw/api/advanced/v2/Metro/Station/NearBy",
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
        return MetroNear.fromJson(response.data);
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
  Future<Object> getBusNearByBus(double Lon,double Lat) async{
    try{
      Response response = await _dio.get(
        "https://tdx.transportdata.tw/api/advanced/v2/Bus/RealTimeByFrequency/NearBy?%24top=30&%24spatialFilter=nearby%28$Lat%2C%20$Lon%2C%20500%29&%24format=JSON",
        options: Options(
          headers: { "authorization": "Bearer $_accesstoken",
            "Content-Encoding": "br,gzip"
          },
        )
      );
      if(response.statusCode == 200){
        return BusNearByBusFromJson(response.data);
      }else{
        throw Exception("Failed to get bus route");
      }
    }
    on DioException catch (e){
      if(e.response?.statusCode == 401){
        await getToken();
        return getBusNearByBus(Lon, Lat);
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
        return BusPositionFromJson(response.data);
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