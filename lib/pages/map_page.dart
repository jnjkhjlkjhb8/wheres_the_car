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
  late GoogleMapController _mapController;
  final Set<Marker> _markers = {};
  String temp = "qbpwCmwlcVc@sBiF~AeAeFrBoApG_Gb@{SI}DaEgRKeB}B}Jc@eDg@yHIeGQg@KuDaB}TaMii@`PaKhFgGpBeBfEqCrBaArFfGBd@ZC@_@fAsA~AKl@FbDd@lAr@pEbHh@XzBR`BjAd@Cn@g@Fc@k@gA";
  @override
  //List<Polyline> loc =PolylinePoints.decodePolyline(temp);
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Test'),
      ),
      body: GoogleMap(
        onMapCreated: (controller) {
          _mapController = controller;
        },
        initialCameraPosition: const CameraPosition(
          target: LatLng(25.033964, 121.564468), // Taipei 101
          zoom: 14,
        ),
        markers: _markers,
      ),
    );
  }
}