import 'package:flutter/material.dart';
import 'package:flutter_polyline_points/flutter_polyline_points.dart';
import 'package:google_maps_flutter/google_maps_flutter.dart';
import 'package:google_maps_utils/google_maps_utils.dart';

class MapPage extends StatefulWidget {
  const MapPage({super.key});

  @override
  State<MapPage> createState() => _MapPageState();
}
class _MapPageState extends State<MapPage> {
  static const String _Maps_API = String.fromEnvironment("GOOGLE_MAPS_API_KEY");
  late GoogleMapController _mapController;
  final Set<Marker> _markers = {};
  final Set<Polyline> _polylines = {};
  Set<Polyline> parsePolygon(String s){
    s = s.replaceAll("LINESTRING(", "").replaceAll(")", "");
    final p = s.split(',');
    List<LatLng> res = [];
    for (var i in p) {
      final temp = i.trim().split(" ");
      if (temp.length >= 2) {
        double lag = double.parse(temp[1]);
        double lon = double.parse(temp[0]);
        res.add(LatLng(lag, lon));
      }
    }
    return {
      Polyline(
      polylineId: const PolylineId("polyline"),
      points: res,
      color: Colors.blue,
      width: 2,
    )};
  }
  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Test'),
      ),
      body: GoogleMap(
        onMapCreated: (controller) {
          _mapController = controller;
          if(_polylines.isNotEmpty) {
            _mapController.animateCamera(
              CameraUpdate.newLatLng(_polylines.first.points.first)
            );
          }
        },
        initialCameraPosition: const CameraPosition(
          target: LatLng(24.99129, 121.31207),
          zoom: 14,
        ),
        markers: _markers,
        polylines: parsePolygon("LINESTRING(121.31207 24.99129,121.31265 24.99147,121.31217 24.99264,121.31332 24.99299,121.31372 24.99241,121.31500 24.99104,121.31834 24.99086,121.31929 24.99091,121.32237 24.99188,121.32288 24.99194,121.32479 24.99257,121.32562 24.99275,121.32719 24.99295,121.32850 24.99300,121.32870 24.99309,121.32961 24.99315,121.33312 24.99364,121.33989 24.99589,121.34182 24.99316,121.34314 24.99199,121.34365 24.99142,121.34438 24.99042,121.34471 24.98984,121.34339 24.98862,121.34320 24.98860,121.34322 24.98846,121.34338 24.98845,121.34380 24.98809,121.34386 24.98761,121.34382 24.98738,121.34363 24.98656,121.34337 24.98617,121.34191 24.98512,121.34178 24.98491,121.34168 24.98429,121.34130 24.98380,121.34132 24.98361,121.34152 24.98337,121.34170 24.98333,121.34206 24.98355)"),
      ),
    );
  }
}