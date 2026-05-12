import 'package:flutter/material.dart';
import 'package:google_maps_flutter/google_maps_flutter.dart';
import 'package:geolocator/geolocator.dart';
import '../api/main.dart';
import 'dart:async';
import 'package:intl/intl.dart';
import 'package:dart_geohash/dart_geohash.dart';
import '../utility/markermaker.dart';
import 'package:smooth_sheets/smooth_sheets.dart';

class MapPage extends StatefulWidget {
  const MapPage({super.key});
  @override
  State<MapPage> createState() => _MapPageState();
}
class _MapPageState extends State<MapPage> with TickerProviderStateMixin{
  static const String _Maps_API = String.fromEnvironment("GOOGLE_MAPS_API_KEY");
  GoogleMapController? _mapController;
  final Set<AdvancedMarker> _markers = {};
  final Set<Polyline> _polylines = {};
  late AnimationController _bottomsheetanimationController;
  late AnimationController dropdownanimation;
  late Animation dropanimation;
  Map<String, BitmapDescriptor> _icons = {};
  List<dynamic>? _busData;
  Map<String,String>? _busname;
  Position? _position;
  int range = 500;
  double zoom = 16.0;
  String? _selected;
  Map<String, List<dynamic>> merge = {};
  List<dynamic> bus = [];
  List<dynamic> mrt = [];
  List<dynamic> bike = [];
  bool init = false;
  double animeoffset = 0.0;
  void Merge() {
    merge = {};
    for (var item in bus) {
      String key = item.StationName['Zh_tw'];
      if (!merge.containsKey(key)) {
        merge[key] = [];
      }
      merge[key]!.add(item);
    }
  }
  @override
  void initState() {
    super.initState();
    _bottomsheetanimationController = BottomSheet.createAnimationController(this);
    _bottomsheetanimationController.duration = Duration(milliseconds: 300);
    dropdownanimation = AnimationController(vsync: this, duration: Duration(milliseconds: 500));
    dropanimation = Tween<double>(begin: -1.0,end: 0.0).animate(
      CurvedAnimation(parent: dropdownanimation, curve: Curves.bounceOut),
    )..addListener((){
      setState(() {
        animeoffset = dropanimation.value;
      });
      buildmarker();
    });
  }
  @override
  void dispose() {
    _bottomsheetanimationController.dispose();
    super.dispose();
  }
  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    if (!init) {
      init = true;
      initicons();
    }
  }
  Future<void> initicons() async {
    final icons = await Markermaker().loadicons();
    if(!mounted) return;
    setState(() {
      _icons = icons;
      init = true;
    });
    if (bus.isNotEmpty || bike.isNotEmpty || mrt.isNotEmpty) buildmarker();
  }
  void _down(dynamic data){
    setState(() {
      _selected = data.StationUID;
      _busData = merge[data.StationName['Zh_tw']];
      _busname = data.StationName;
    });
    dropdownanimation.forward(from: 0.0);
  }
  void _up() async{
    if(_selected != null){
      await dropdownanimation.reverse();
      setState(() {
        _selected = null;
      });
      buildmarker();
    }
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
  void buildmarker(){
    setState(() {
      _markers.clear();
      for (var i in bus) {
        _markers.add(createMarker(i, _icons['Bus'],_icons['RD']));
      }
      for (var i in mrt){
        _markers.add(createMarker(i, _icons['BL'],_icons['BD']));
      }
      for (var i in bike){
        _markers.add(createMarker(i, _icons['Bike'],_icons['GD']));
      }
    });
  }
  AdvancedMarker createMarker(dynamic data,BitmapDescriptor? icon,BitmapDescriptor? dot){
    final String UID = data.StationUID;
    final bool selected = _selected == UID;
    BitmapDescriptor ficon;
    Offset anchor = selected ? Offset(0.5,1.0- animeoffset) : const Offset(0.5,0.5);
    if (selected) {
      ficon = _icons['Pegman'] ?? BitmapDescriptor.defaultMarkerWithHue(BitmapDescriptor.hueYellow);
    }
    else if(zoom >= 17.0){
      ficon = icon!;
    }
    else {
      ficon = dot!;
    }
    return AdvancedMarker(
      markerId: MarkerId(UID),
      position: LatLng(data.PositionLat, data.PositionLon),
      icon: ficon,
      anchor: anchor,
      onTap: () {
        setState(() {
          if (_selected == UID) return;
          if (_selected != null){
            dropdownanimation.reverse().then((_){
              _down(data);
            });
          }
          else {
            _down(data);
          }
        });
        buildmarker();
        _mapController!.animateCamera(CameraUpdate.newLatLng(LatLng(data.PositionLat, data.PositionLon)));
      },
    );
  }
  Future<void> update() async {
    try {
      Position pos = await _getCurrentLocation();
      _mapController?.animateCamera(
        CameraUpdate.newLatLngZoom(
          LatLng(pos.latitude, pos.longitude),
          16,
        ),
      );
      debugPrint("Current Position: ${pos.latitude}, ${pos.longitude}");
      final List<dynamic> busData = await Tdx().getBusNearByStation(pos.longitude, pos.latitude, range);
      final List<dynamic> mrtData = await Tdx().getMetroNear(pos.longitude, pos.latitude, range);
      final List<dynamic> bikeData = await Tdx().getBikeNear(pos.longitude, pos.latitude, range);
      setState(() {
        _position = pos;
        bus = busData;
        mrt = mrtData;
        bike = bikeData;
        Merge();
        buildmarker();
      });
    }
    catch (e) {
      debugPrint("Error : $e");
    }
  }
  @override
  Widget build(BuildContext context) {
    final systemUiInsets = MediaQuery.of(context).padding;
    if (_position == null){
      return Scaffold(
        body: Center(
          child: CircularProgressIndicator(),
        ),
      );
    }
    return Stack(
      children:[
       Scaffold(
        body: Stack(
          children: [
            GoogleMap(
              myLocationEnabled: true,
              zoomControlsEnabled: false,
              myLocationButtonEnabled: false,
              padding: const EdgeInsets.only(bottom: 200),
              initialCameraPosition: CameraPosition(
                target: LatLng(25.0339, 121.5646),
                zoom: 16,
              ),
              onCameraMove: (position){
                zoom = position.zoom;
              },
              onCameraIdle: (){
                buildmarker();
              },
              onTap: (latlng) async{
                if(_selected != null) {
                  await dropdownanimation.reverse();
                  setState(() {
                    _selected = null;
                    _busData = null;
                    _busname = null;
                  });
                  buildmarker();
                }
              },
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
            SheetViewport(
              child: _Sheet(
                merge: merge,
                position: _position!,
                JumpIn: (data){
                  final String temp = GeoHasher().encode(data[0].PositionLon, data[0].PositionLat, precision: 10);
                  int count = 0,len = temp.length > data[0].GeoHash.length ? data[0].GeoHash.length : temp.length;
                  for (int i = 0; i < len; i++) {
                    if (temp[i] == data[0].GeoHash[i]) {
                      count++;
                    }
                    else {
                      break;
                    }
                  }
                  _mapController?.animateCamera(
                    CameraUpdate.newLatLngZoom(
                      LatLng((_position!.latitude + data[0].PositionLat) / 2, (_position!.longitude + data[0].PositionLon) / 2),
                      count >= 7 ? 17.0 : count == 6 ? 16.0 : count == 5 ? 14.0 : count == 4 ? 12.0 : count == 3 ? 10.0 : count == 2 ? 8.0 : count == 1 ? 6.0 : 5.0,
                    ),
                  );
                },
                range: range,
                onRangeChanged: (data){
                  setState(() {
                    range = data;
                  });
                  update();
                },
              ),
            )
          ],
        ),
      ),
    ]);
  }
}
class _Sheet extends StatelessWidget { // https://github.com/fujidaiti/smooth_sheets/blob/main/example/lib/tutorial/imperative_paged_sheet.dart
  final Map<String,List<dynamic>> merge;
  final Position position;
  final Function(dynamic) JumpIn;
  final Function(dynamic) onRangeChanged;
  final int range;
  const _Sheet({
    required this.merge,
    required this.position,
    required this.JumpIn,
    required this.onRangeChanged,
    required this.range,
  });
  @override
  Widget build(BuildContext context) {
    final nestedNavigator = Navigator(
      onGenerateInitialRoutes: (navigator, initialRoute) {
        return [
          PagedSheetRoute(
            builder: (context) {
              return _StationSheet(
                merge: merge,
                position: position,
                JumpIn: JumpIn,
                onRangeChanged: onRangeChanged,
                range: range,
              );
            },
          ),
        ];
      },
    );
    return PagedSheet(
      decoration: MaterialSheetDecoration(
        size: SheetSize.stretch,
        borderRadius: BorderRadius.circular(20),
        clipBehavior: Clip.antiAlias,
        color: Theme.of(context).colorScheme.surface,
      ),
      navigator: nestedNavigator,
    );
  }
}
class _StationSheet extends StatelessWidget{
  final Map<String,List<dynamic>> merge;
  final Position position;
  final Function(dynamic) JumpIn;
  final Function(dynamic) onRangeChanged;
  final int range;
  const _StationSheet({
    required this.merge,
    required this.position,
    required this.JumpIn,
    required this.onRangeChanged,
    required this.range,
  });
  @override
  Widget build(BuildContext context) {
    void NavigateToBus(BuildContext context,String Name) {
      final route = PagedSheetRoute(
          scrollConfiguration: SheetScrollConfiguration(),
          builder: (context) => BusEstimate(
            stationName: merge[Name]![0].StationName,
            stations: merge[Name]!,
            onBack: () => Navigator.of(context).pop(),
            scrollController: ScrollController(),
          )
      ,);
      Navigator.of(context).push(route
      );
    }
    Widget busstops(String name) {
      final data = merge[name];
      if (data == null || data.isEmpty == null) return Container();
      return ListTile(
        title: Text(data[0].StationName['Zh_tw'] ?? name),
        subtitle: Text("約${Geolocator.distanceBetween(position.latitude, position.longitude, data[0].PositionLat, data[0].PositionLon).toStringAsFixed(2)}公尺"),
        trailing: IconButton(
          onPressed: () => JumpIn(data),
          icon: Icon(Icons.location_searching_rounded),
        ),
        onTap: () {
          NavigateToBus(context,name);
        },
      );
    }
    return LayoutBuilder(
        builder: (context, setBottomSheetState) {
          List <String> temp = merge.keys.toList();
          List<Widget> array = [];
          temp.sort((a, b) {
            double distA = Geolocator.distanceBetween(position.latitude, position.longitude, merge[a]![0].PositionLat, merge[a]![0].PositionLon);
            double distB = Geolocator.distanceBetween(position.latitude, position.longitude, merge[b]![0].PositionLat, merge[b]![0].PositionLon);
            return distA.compareTo(distB);
          });
          ColorScheme colorscheme = Theme.of(context).colorScheme;
          return DefaultTabController(
              length: 3,
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
                    Expanded(
                        child: Column(
                            children: [
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
                                      onSelectionChanged: (Set<int> newSelection) async {
                                        onRangeChanged(newSelection.first);
                                      },
                                    ),
                                  ],
                                ),
                              ),
                              TabBar(
                                labelColor: colorscheme.primary,
                                tabs: [
                                  Tab(text: '公車'),
                                  Tab(text: '捷運 / 輕軌'),
                                  Tab(text: '公共自行車'),
                                ],
                              ),
                              Expanded(
                                child: TabBarView(
                                  children: [
                                    merge.isEmpty
                                        ? Center(child: Text('附近沒有公車站牌'))
                                        : ListView.builder(
                                            physics: AlwaysScrollableScrollPhysics(),
                                            itemCount: temp.length,
                                            itemBuilder: (context, index) {
                                                return busstops(temp[index]);
                                              }
                                            ),
                                    Center(child: Text('捷運 / 輕軌資訊')),
                                    Center(child: Text('公共自行車資訊')),
                                  ],
                                ),
                              ),
                            ]
                        )
                    ),
                  ]
              )
          );
        },
    );
  }
}
class BusEstimate extends StatefulWidget{
  final Map<String,String> stationName;
  final List<dynamic> stations;
  final VoidCallback onBack;
  final ScrollController scrollController;
  const BusEstimate({
    super.key,
    required this.stationName,
    required this.stations,
    required this.onBack,
    required this.scrollController,
  });
  @override
  State<BusEstimate> createState() => _BusEstimateState();
}
class _BusEstimateState extends State<BusEstimate> with SingleTickerProviderStateMixin{
  Timer? _timer;
  bool refresh = false;
  bool fetch = false;
  final Map<String,List<dynamic>> _datas = {};
  late AnimationController _animationController;
  @override
  void initState() {
    super.initState();
    _animationController = AnimationController(vsync: this,duration: const Duration(seconds: 15));
    _fetchData();
    update();
  }
  void update() {
    _animationController.repeat();
    _timer = Timer.periodic(const Duration(seconds: 15), (timer) {
      _fetchData();
    });
  }
  @override
  void dispose() {
    _animationController.dispose();
    _timer?.cancel();
    super.dispose();
  }
  Future<void> _fetchData() async {
    if(!mounted || fetch) return;
    fetch = true;
    setState(() => refresh = true);
    try {
      await Future.wait(widget.stations.map((i) async {
        List<dynamic>data;
        if (i.StationUID.toString().substring(0,3) != "THB"){
          data = await Tdx().getBusEstimateByStation(i.LocationCityCode, i.StationID);
        }
        else {
          data = await Tdx().getInterBusEstimateByStation(i);
        }
        data.sort((a, b) {
          if (a.EstimateTime != null && b.EstimateTime != null) {
            return a.EstimateTime!.compareTo(b.EstimateTime!);
          }
          String t1 = a.NextBusTime?.toString() ?? "9999-12-31T23:59:59";
          String t2 = b.NextBusTime?.toString() ?? "9999-12-31T23:59:59";
          if (a.StopStatus == 2 || a.StopStatus == 3 || a.StopStatus == 4) return 1;
          if (b.StopStatus == 2 || b.StopStatus == 3 || b.StopStatus == 4) return -1;
          return t1.compareTo(t2);
        });
        _datas[i.StationID] = data;
      }));
    } catch (e) {
      if(mounted) refresh = false;
      debugPrint("e2: $e");
    } finally{
      if(mounted) setState(() => refresh = false);
      fetch = false;
    }
  }
  @override
  Widget buildlisttile(dynamic estimate, dynamic colorsceme){
    int? EstimateTime = estimate?.EstimateTime;
    int? status = estimate?.StopStatus;
    Color color;
    dynamic text;
    if (status == 3){
      text = Text("末班車已過",style: TextStyle(fontSize: 13,fontWeight: FontWeight.bold));
      color = Colors.grey;
    }
    else if (status == 2){
      text = Text("交管不停靠",style: TextStyle(fontSize: 13));
      color = Colors.grey;
    }
    else if (status == 1){
      text = Text(DateFormat("HH:mm").format(estimate?.NextBusTime?.toLocal()),style: TextStyle(fontSize: 18,fontWeight: FontWeight.bold));
      color = Colors.grey;
    }
    else if (status == 4){
      text = Text("今日未營運",style: TextStyle(fontSize: 13));
      color = Colors.grey;
    }
    else if (status == 255 || EstimateTime == null){
      text = Text("資料不可用",style: TextStyle(fontSize: 13));
      color = Colors.grey;
    }
    else if (EstimateTime < 60){
      text = Text("進站中",style: TextStyle(fontSize: 15,fontWeight: FontWeight.bold));
      color = Colors.red.shade900;
    }
    else if (EstimateTime <= 120){
      text = Text("即將進站",style: TextStyle(fontSize: 15,fontWeight: FontWeight.bold));
      color = colorsceme.primary;
    }
    else{
      text = RichText(
          text: TextSpan(
            text: "${(EstimateTime/60).floor()}",
            style: TextStyle(fontSize: 20,fontWeight: FontWeight.bold,color: Colors.black),
            children: [
              WidgetSpan(child: const SizedBox(width: 5)),
              TextSpan(
                text: "分",
                style: TextStyle(fontSize: 12, color: Colors.grey[600], fontWeight: FontWeight.normal),
              ),
            ],
          )
      );
      color = colorsceme.secondary;
    }
    return ListTile(
      contentPadding: EdgeInsets.symmetric(horizontal: 20, vertical: 0),
      leading: Container(
          width: 75,
          height: 30,
          decoration: BoxDecoration(
            borderRadius: BorderRadius.circular(6.7),
            border: Border.all(color: color, width: 1.67),
          ),
          alignment: Alignment.center,
          child: text
      ),
      title: Text(estimate.SubRouteName["Zh_tw"],style: TextStyle(fontWeight: FontWeight.bold)),
      subtitle: Text("往 ${estimate.DestinationStop ?? "Error"}",style: TextStyle(color: Colors.grey)),
    );
  }
  @override
  Widget build(BuildContext context) {
    final ColorScheme colorsheme = Theme.of(context).colorScheme;
    return DefaultTabController(
        length: widget.stations.length,
        child: Scaffold(
          backgroundColor: Colors.transparent,
          appBar: AppBar(
            primary: false,
            automaticallyImplyLeading: false,
            title: Row(
              children: [
                IconButton(
                  icon: Icon(Icons.arrow_back_ios_new_rounded),
                  onPressed: widget.onBack,
                ),
                Expanded(
                  child: Text(
                    widget.stationName['Zh_tw'] ?? 'Error',
                    style: TextStyle(fontSize: 20, fontWeight: FontWeight.bold),
                  ),
                )
              ],
            ),
            bottom: PreferredSize(
              preferredSize: Size.fromHeight(50.0),
              child: Stack(
                alignment: Alignment.bottomCenter,
                children: [
                  TabBar(
                    labelColor: colorsheme.primary,
                    isScrollable: widget.stations.length > 3,
                    tabs: List.generate(widget.stations.length,(index){
                      return Tab(text: widget.stations[index].Bearing ?? String.fromCharCode(65 + index));
                    }),
                  ),
                  AnimatedBuilder(animation: _animationController, builder:
                      (context, child) {
                    return LinearProgressIndicator(
                      value: refresh ? null : (1.0-_animationController.value),
                      color: refresh ? colorsheme.secondary : colorsheme.primary,
                      minHeight: 3.0,
                      backgroundColor: Colors.transparent,
                    );
                  }
                  ),
                ],
              ),
            ),
          ),
          body: TabBarView(
            children: List.generate(widget.stations.length,(index) {
              final List<dynamic> route = _datas[widget.stations[index].StationID] ?? [];
              return ListView.builder(
                physics: const AlwaysScrollableScrollPhysics(),
                controller: widget.scrollController,
                itemCount: route.length,
                itemBuilder: (context, i) {
                  return buildlisttile(route[i], colorsheme);
                },
              );
            }),
          ),
        )
    );
  }
}