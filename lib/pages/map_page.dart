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

      ),
    );
  }
}