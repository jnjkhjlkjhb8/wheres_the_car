import 'package:flutter/material.dart';
import 'package:flutter_polyline_points/flutter_polyline_points.dart';
import 'package:google_maps_flutter/google_maps_flutter.dart';
import 'package:google_maps_utils/google_maps_utils.dart';
import 'package:geolocator/geolocator.dart';
import '../api/main.dart';
import 'package:dart_geohash/dart_geohash.dart';

class MapPage extends StatefulWidget {
  const MapPage({super.key});

  @override
  State<MapPage> createState() => _MapPageState();
}
class _MapPageState extends State<MapPage> with TickerProviderStateMixin{
  static const String _Maps_API = String.fromEnvironment("GOOGLE_MAPS_API_KEY");
  late GoogleMapController _mapController;
  final Set<Marker> _markers = {};
  final Set<Polyline> _polylines = {};
  late AnimationController _animationController;
  Position? _position;
  Map<String, List<dynamic>> merge = {};
  List<dynamic> bus = [];
  List<dynamic> mrt = [];
  List<dynamic> bike = [];

  void Merge() {
    merge = {};
    for (var item in bus) {
      String key = item.StationUID;
      if (!merge.containsKey(key)) {
        merge[key] = [];
      }
      merge[key]!.add(item);
    }
  }
  int range = 500;
  @override
  void initState() {
    super.initState();
    _animationController = BottomSheet.createAnimationController(this);
    _animationController.duration = Duration(milliseconds: 300);
  }
  void dispose() {
    _animationController.dispose();
    super.dispose();
  }
  Future<Position> _getCurrentLocation() async {
    LocationPermission permission;
    bool yes = await Geolocator.isLocationServiceEnabled();
    permission = await Geolocator.checkPermission();
    if (permission == LocationPermission.denied) {
      permission = await Geolocator.requestPermission();
      if (permission == LocationPermission.denied) {
        return Future.error('Location permissions are denied');
      }
    }
    if (permission == LocationPermission.deniedForever || !yes) {
      return Future.error('Location permissions are denied, we cannot request permissions.');
    } else {
    return Geolocator.getCurrentPosition(desiredAccuracy: LocationAccuracy.high);}
  }

  void update() async {
    try {
      Position pos = await _getCurrentLocation();
      Object busData = await Tdx().getBusNearByStation(pos.longitude, pos.latitude, range);
      Object mrtData = await Tdx().getMetroNear(pos.longitude, pos.latitude, range);
      Object bikeData = await Tdx().get(pos.longitude, pos.latitude
      setState(() {
        _position = pos;
        if (busData is List<dynamic>) {
          bus = busData;
        }
        Merge();
      });
      _mapController.animateCamera(
        CameraUpdate.newLatLngZoom(
          LatLng(pos.latitude, pos.longitude),
          16,
        ),
      );
    } catch (e) {
      debugPrint("Error updating location/bus: $e");
    }
  }
  @override
  Widget build(BuildContext context) {
    final ColorScheme colorsheme = Theme.of(context).colorScheme;
    Widget Stationtile(String name) {
      final data = merge[name];
      if (data == null || data.isEmpty || _position == null) return Container();
      final String temp = GeoHasher().encode(data[0].PositionLat, data[0].PositionLon, precision: 10);
      return ListTile(
        title: Text(data[0].StationName['Zh_tw'] ?? name),
        subtitle: Text("約${Geolocator.distanceBetween(_position!.latitude, _position!.longitude, data[0].PositionLat, data[0].PositionLon).toStringAsFixed(2)}公尺"),
        leading: IconButton(
          onPressed: () {
            int count = 0;
            int len = temp.length > data[0].GeoHash.length ? data[0].GeoHash.length : temp.length;
            for (int i = 0; i < len; i++) {
              if (temp[i] == data[0].GeoHash[i]) {
                count++;
              } else {
                break;
              }
            }
            _mapController.animateCamera(
              CameraUpdate.newLatLngZoom(
                LatLng((_position!.latitude + data[0].PositionLat) / 2, (_position!.longitude + data[0].PositionLon) / 2),
                count >= 7 ? 18.0 : count == 6 ? 16.0 : count == 5 ? 14.0 : 12.0,
              ),
            );
          },
          icon: Icon(Icons.location_searching_rounded),
        ),
      );
    }
    return Scaffold(
      bottomSheet: BottomSheet(
        animationController: _animationController,
        backgroundColor: colorsheme.surface,
        enableDrag: true,
        onClosing: () {},
        builder: (context) {
          return DefaultTabController(
            length: 3,
            child: DraggableScrollableSheet(
              minChildSize: 0.1,
              initialChildSize: 0.2,
              expand: false,
              builder: (context, scrollController) {
                return Container(
                  child: Column(
                    children: [
                      Container(
                        height: 5,
                        width: 50,
                        margin: EdgeInsets.only(top: 8.0),
                        decoration: BoxDecoration(
                          color: Colors.grey,
                          borderRadius: BorderRadius.circular(10),
                        ),
                      ),
                      Padding(
                        padding: EdgeInsets.symmetric(vertical: 8.0, horizontal: 16.0),
                        child: Row(
                          mainAxisAlignment: MainAxisAlignment.spaceBetween,
                          children: [
                            Text(
                             '何近也',
                              style: TextStyle(fontSize: 20, fontWeight: FontWeight.bold),
                            ),
                            SegmentedButton<int>(
                              showSelectedIcon: false,
                              segments: const [
                                ButtonSegment(value: 500, label: Text('500公尺')),
                                ButtonSegment(value: 750, label: Text('750公尺')),
                                ButtonSegment(value: 1000, label: Text('1000公尺')),
                              ],
                              selected: <int>{range},
                              onSelectionChanged: (Set<int> newSelection) {
                                setState(() {
                                  range = newSelection.first;
                                  update();
                                });
                              },
                            ),
                          ],
                        ),
                      ),
                      TabBar(
                        labelColor: colorsheme.primary,
                        tabs: [
                          Tab(text: '公車'),
                          Tab(text: '捷運 / 輕軌'),
                          Tab(text: '公共自行車'),
                        ],
                      ),
                      Expanded(
                        child: TabBarView(
                          children: [
                            Builder(
                              builder: (context) {
                                List<Widget> array = [];
                                for (String i in merge.keys) array.add(Stationtile(i));
                                return ListView(
                                  controller: scrollController,
                                  children: array,
                                );
                              }
                            ),
                            Center(child: Text('捷運 / 輕軌資訊')),
                            Center(child: Text('公共自行車資訊')),
                          ],
                        ),
                      ),
                    ],
                  )
                );
              },
            )
          );
        },
      ),
      body: Stack(
        children: [
          GoogleMap(
            myLocationButtonEnabled: true,
            myLocationEnabled: true,
            zoomControlsEnabled: false,
            padding: const EdgeInsets.only(bottom: 200),
            initialCameraPosition: CameraPosition(
              target: LatLng(0, 0),
              zoom: 5,
            ),
            markers: _markers,
            polylines: _polylines,
            onMapCreated: (GoogleMapController controller) {
              _mapController = controller;
              update();
            },
          ),
          Positioned(
            bottom: 220,
            right: 20,
            child: FloatingActionButton(
              onPressed: update,
              child: Icon(Icons.my_location),
            ),
          ),
        ],
      ),
    );
  }
}