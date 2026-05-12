import 'dart:async';
import 'package:bus/data/DailyRoute.dart';
import 'package:flutter/material.dart';
import 'package:animated_text_kit/animated_text_kit.dart';
import '../utility/Favorite.dart';
import '../api/main.dart';
import '../utility/update_route.dart';
import 'package:collection/collection.dart';
import 'package:intl/intl.dart';
import 'dart:math';
final Map<String, String> _cites = {
    "Taipei": "台北市",
    "NewTaipei": "新北市",
    "Taoyuan": "桃園市",
    "Taichung": "台中市",
    "Tainan": "台南市",
    "Kaohsiung": "高雄市",
    "Keelung": "基隆市",
    "Hsinchu": "新竹市",
    "HsinchuCounty": "新竹縣",
    "MiaoliCounty": "苗栗縣",
    "ChanghuaCounty": "彰化縣",
    "NantouCounty": "南投縣",
    "YunlinCounty": "雲林縣",
    "ChiayiCounty": "嘉義縣",
    "Chiayi": "嘉義市",
    "PingtungCounty": "屏東縣", 
    "YilanCounty": "宜蘭縣",
    "HualienCounty": "花蓮縣",
    "TaitungCounty": "台東縣",
    "KinmenCounty": "金門縣",
    "PenghuCounty": "澎湖縣",
    "LienchiangCounty": "連江縣",
    "InterCity": "公路客運"
};
class HomePage extends StatefulWidget{
  const HomePage({super.key});
  @override
  State<HomePage> createState() => _HomePageState();
}
class _HomePageState extends State<HomePage> {
  final UpdateRoute fetchRoutes = UpdateRoute();
  final SearchController _searchController = SearchController();
  final List<String> Greeting = [
    "Ciallo～(∠・ω< )⌒☆",
    "哈基米豆南北路豆",
    "你是小男娘",
  ];
  List<AnimatedText> get animatedTexts {
    final hour = DateTime.now().hour;
    if (hour < 12) {
      return [TypewriterAnimatedText(Greeting[0], textStyle: TextStyle(fontSize: 25, fontWeight: FontWeight.bold,color: Colors.black),speed: Duration(milliseconds: 100))];
    } else if (hour < 18) {
      return [TypewriterAnimatedText(Greeting[1], textStyle: TextStyle(fontSize: 25, fontWeight: FontWeight.bold,color: Colors.black),speed: Duration(milliseconds: 100),)];
    } else {
      return [TypewriterAnimatedText(Greeting[2], textStyle: TextStyle(fontSize: 25, fontWeight: FontWeight.bold,color: Colors.black),speed: Duration(milliseconds: 100), )];
    }
  }
  late Future<List<Favorite>> _favorites;
  int currentPage = 0;
  void initState(){
    super.initState();
    _favorites = getFavorites();
  }
  @override
  Widget build(BuildContext context){
    final colorscheme = Theme.of(context).colorScheme;
      return Scaffold(
        body: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Container(
              height: 100,
              width: double.infinity,
              decoration: BoxDecoration(
                color: colorscheme.primaryContainer,
                borderRadius: BorderRadius.only(bottomRight: Radius.circular(33.5), bottomLeft: Radius.circular(33.5)),
              ),
              child: Padding(
                padding: const EdgeInsets.fromLTRB(30, 50, 24, 0),
                child: Center(
                  child: SizedBox(
                      height: 50,
                      child: AnimatedTextKit(
                        animatedTexts: animatedTexts,
                        repeatForever: true,
                      ),
                    )
                ),
              )
            ),
            Padding(
              padding: const EdgeInsets.only(left:15.0,top: 10),
              child: Column(
                children: [
                  Text("我的最愛",style: TextStyle(color: Colors.grey[600],fontWeight: FontWeight.bold,fontSize: 18,)),
                ],
              ),
            ),
            ConstrainedBox(
              constraints: BoxConstraints(maxHeight: 200),
              child: FutureBuilder<List<Favorite>>(
                future: _favorites,
                builder: (context, snapshot) {
                  if (snapshot.connectionState == ConnectionState.waiting) {
                    return const Center(child: CircularProgressIndicator());
                  } else if (snapshot.hasError) return const Center(child: Text('發生錯誤，無法載入最愛路線',style: TextStyle(fontWeight: FontWeight.bold)));
                  else if (snapshot.data!.isEmpty) return const Center(child: Text('尚未新增過任何東西',style: TextStyle(fontWeight: FontWeight.bold)));
                  else {
                    final favorites = snapshot.data!;
                    return ListView.builder(
                      shrinkWrap: true,
                      itemCount: favorites.length,
                      itemBuilder: (context, index) {
                        final favorite = favorites[index];
                        return Dismissible(
                          key: Key(favorite.route),
                          direction: DismissDirection.endToStart,
                          background: Container(
                            color: Colors.red,
                            alignment: Alignment.centerRight,
                            padding: EdgeInsets.symmetric(horizontal: 20),
                            child: Icon(Icons.delete, color: Colors.white),
                          ),
                          onDismissed: (direction) async {
                            await delFavorites(index, favorites);
                            setState(() {});
                          },
                          child: ListTile(
                            title: Text(favorite.route),
                            subtitle: Text('${favorite.StartStaion} → ${favorite.EndStation}'),
                          ),
                        );
                      },
                    );
                  }
                }
              )
            ),
            const SizedBox(height: 20,child: Divider(thickness: 1.67,)),
            Padding(
              padding: EdgeInsets.all(15),
              child: SearchAnchor(
                searchController: _searchController,
                builder: (context, controller){
                  return SearchBar(
                    controller: controller,
                    hintText: "搜尋路線",
                    leading: Icon(Icons.search),
                    onTap: () => controller.openView(),
                  );
                },
                suggestionsBuilder: (context,controller) async {
                  if (allRoutes.isEmpty) await fetchRoutes.fetchRoutes();
                  final input = controller.text.trim().toUpperCase();
                  Map<String,dynamic> recentList = await recent().get();
                  if (input.isEmpty){
                    if (recentList.isEmpty) return [const ListTile(title: Text("沒有最近搜尋紀錄"))];
                    List<Routes> recentRoutes = allRoutes.where((route) => recentList.containsKey(route.RouteUID)).toList();
                    recentRoutes.sort((a, b) => recentList[b.RouteUID].compareTo(recentList[a.RouteUID]));
                    return recentRoutes.map((route) => ListTile(
                      title: Text.rich(
                        TextSpan(
                          text: route.SubRouteName,
                          style: TextStyle(fontWeight: FontWeight.bold),
                          children: [
                            WidgetSpan(child: const SizedBox(width: 5)),
                            TextSpan(
                              text: _cites[route.City],
                              style: TextStyle(fontSize: 12, color: Colors.grey[600], fontWeight: FontWeight.normal),
                            ),
                          ],
                        ),
                      ),
                      subtitle: Text("${route.DepartureStopNameZh} ⇌ ${route.DestinationStopNameZh}"),
                      trailing: (
                        IconButton(
                          icon: Icon(Icons.delete,color: Colors.red[400]),
                          onPressed: (){
                            showDialog(
                              context: context,
                              builder: (context) => AlertDialog(
                                title: Text("確定要刪除這項紀錄嗎？",style: TextStyle(fontWeight: FontWeight.bold)),
                                content: Column(
                                  mainAxisSize: MainAxisSize.min,
                                  crossAxisAlignment: CrossAxisAlignment.start,
                                  children: [
                                    Text.rich(
                                      TextSpan(
                                        text: route.SubRouteName,
                                        style: TextStyle(fontWeight: FontWeight.bold,fontSize: 20),
                                        children: [
                                          WidgetSpan(child: const SizedBox(width: 5)),
                                          TextSpan(
                                            text: _cites[route.City],
                                            style: TextStyle(fontSize: 12, color: Colors.grey[600], fontWeight: FontWeight.normal),
                                          ),
                                        ],
                                      ),
                                    ),
                                    Text("${route.DepartureStopNameZh} ⇌ ${route.DestinationStopNameZh}"),
                                  ],
                                ),
                                actions: [
                                  FilledButton.tonal(
                                    onPressed: () => Navigator.pop(context),
                                    child: Text("取消"),
                                  ),
                                  FilledButton(
                                    onPressed: () async {
                                      await recent().del(route.RouteUID);
                                      setState(() {});
                                      String temp = controller.text;
                                      controller.text = "${controller.text} ";
                                      controller.text = temp;
                                      Navigator.pop(context);
                                    },
                                    child: Text("確定"),
                                  ),
                                ],
                              ),
                            );
                          },
                        )
                      ),
                      onTap: () {
                        controller.text = route.SubRouteName;
                        recent().add(route.RouteUID);
                        Navigator.push(
                          context,
                          MaterialPageRoute(
                            builder: (context) => BusPage(route: route)
                          )
                        );
                      },
                    )).toList();
                  }
                  final foundRoutes = allRoutes.where((route){
                    if(!route.SubRouteName.contains(input)) {
                      return false;
                    } else if (route.City == "InterCity" && route.SubRouteUID!.endsWith('2')) return false;
                    return route.SubRouteName.contains(input);
                  }).toList();
                  if (foundRoutes.isEmpty) return [const ListTile(title: Text("沒有找到相關路線"))];
                  Map<String, List<Routes>> city = {};
                  List<Widget> Display = [];
                  for (var route in foundRoutes) {
                    city.putIfAbsent(route.City, () => []).add(route);
                  }
                  city.forEach((cityKey, routes) {
                    Display.add(
                      Padding(
                        padding: const EdgeInsets.all(15),
                        child: Text(_cites[cityKey] ?? cityKey,style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold, color: colorscheme.primary)),
                      ),
                    );
                    Display.addAll(
                      routes.map((route) => ListTile(
                        title: Text(route.SubRouteName,style: TextStyle(fontWeight: FontWeight.bold)),
                        subtitle: Text("${route.DepartureStopNameZh} ⇌ ${route.DestinationStopNameZh}"),
                        onTap: () {
                          controller.text = route.SubRouteName;
                          recent().add(route.RouteUID);
                          Navigator.push(
                            context,
                            MaterialPageRoute(
                              builder: (context) => BusPage(route: route)
                            )
                          );
                        },
                      )).toList()
                    );
                  });
                  return Display;
                },
              )
            )
          ]
        )
      );
  }
}
class BusPage extends StatefulWidget{
  final Routes route;
  const BusPage({super.key, required this.route});
  @override
  State<BusPage> createState() => _BusPageState();
}
class _BusPageState extends State<BusPage> with SingleTickerProviderStateMixin{
  bool refresh = false;
  ColorScheme get colorscheme => Theme.of(context).colorScheme;
  late AnimationController _animationController;
  late final Map<String,List<dynamic>> _data = {"estimates": [],"stops": []};
  Timer? _timer;
  int select = 0;
  Map<String,int> time = {};
  final ScrollController ball = ScrollController();
  Widget buildlisttile(dynamic stop, dynamic estimate, dynamic colorsceme,bool first,bool last,dynamic stopschedule){
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
      text = Text(DateFormat("HH:mm").format(estimate?.NextBusTime?.toLocal() ?? ((stopschedule.Timeables as List).where((i) {return i.DepartureTime.toLocal().isAfter(DateTime.now);}),style: TextStyle(fontSize: 18,fontWeight: FontWeight.bold))));
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
      color = colorscheme.secondary;
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
      title: Text(stop.StopName?["Zh_tw"] ?? "",style: TextStyle(fontWeight: FontWeight.bold)),
      trailing: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 16.0,vertical: 0),
        child: Stack(
          alignment: Alignment.center,
          children: [
              Container(
                width: 16,
                decoration: BoxDecoration(
                  color: color,
                  borderRadius: BorderRadius.vertical(top: first?Radius.circular(100): Radius.zero, bottom: last ? Radius.circular(100) : Radius.zero)
                ),
            ),
            Container(
              width: 10,
              height: 10,
              decoration: BoxDecoration(
                color: Colors.white,
                shape: BoxShape.circle,
              ),
            ),
          ],
        )
      ),
    );
  }
  @override
  void initState(){
    super.initState();
    _animationController = AnimationController(vsync: this,duration: const Duration(seconds: 15));
    _fetchData();
    update();
  }
  bool loading = true;
  Future<void> _fetchData() async{
    setState(() {
      refresh = true;
    });
    List<dynamic> estimates =[];
    List<dynamic> stops = [];
    List<dynamic> schedulesstops = [];
    //List<dynamic> schedules = [];
    //List<dynamic> S2S = [];
    try {
      if (widget.route.City == "InterCity"){
        String? temp = widget.route.SubRouteUID;
        String temp2 = "${temp!.substring(0,temp.length-1)}2";
        estimates = await Tdx().getInterBusEstimatedTimeOfArrival(temp,temp2);
        stops = await Tdx().getInterBusStopOfRoute(temp,temp2);
      }
      else{
        estimates = await Tdx().getBusEstimatedTimeOfArrival(widget.route.City, widget.route.RouteUID);
        stops = await Tdx().getBusStopOfRoute(widget.route.City, widget.route.RouteUID);
        schedulesstops = await Tdx().getBusDailyStopTable(widget.route.SubRouteUID!, widget.route.City);
        //schedules = await Tdx().getBusDailyTable(widget.route.SubRouteUID!, widget.route.City);
        //S2S = await Tdx().getBusS2S(widget.route.SubRouteUID!, widget.route.City,widget.route.RouteID!);
      }
      _data["estimates"] = estimates;
      _data["stops"] = stops;
      _data["schedules"] = schedulesstops;
      //_data["S2S"] = S2S;
      //_data["schedules"] = schedules;
      san();
      loading = false;
      return;
    } catch (e) {
      refresh = false;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('error take: $e')),
      );
      rethrow;
    } finally {
      if(mounted) {
        setState(() {
        refresh = false;
      });
      }
      refresh = false;
    }
  }
  Future<void> getdata() async {
    if(refresh || loading) return;
    setState(() {
      refresh = true;
    });
    List<dynamic> estimates =[];
    try {
      if (widget.route.City == "InterCity"){
        String? temp = widget.route.SubRouteUID;
        String temp2 = "${temp!.substring(0,temp.length-1)}2";
        estimates = await Tdx().getInterBusEstimatedTimeOfArrival(temp,temp2);
      }
      else{
        estimates = await Tdx().getBusEstimatedTimeOfArrival(widget.route.City, widget.route.RouteUID);
      }
      if(mounted){
        setState(() {
          _data["estimates"] = estimates;
        });
      }
    } catch (e) {
      refresh = false;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('error take: $e')),
      );
      rethrow;
    } finally {
      if(mounted) {
        setState(() {
        refresh = false;
      });
      }
      refresh = false;
    }
  }
  @override
  void dispose(){
    _animationController.dispose();
    ball.dispose();
    _timer?.cancel();
    super.dispose();
  }
  void update() {
    _animationController.repeat();
    _timer = Timer.periodic(const Duration(seconds: 300000), (timer) {
      getdata();
    });
  }
  int index = 0;
  Widget flip(){ // 參考 https://medium.com/flutter-community/flutter-flip-card-animation-eb25c403f371
    return Container(
      padding: const EdgeInsets.all(8.0),
      child: AnimatedSwitcher(
        duration: Duration(milliseconds: 500),
        transitionBuilder: (Widget child,Animation<double> animation) {
            final temp = Tween(begin: pi, end: 0.0).animate(animation);
            return AnimatedBuilder(
              animation: temp,
              child: child,
              builder: (context, child) {
                final isUnder = temp.value > pi /2;
                var tilt = ((temp.value - 0.5).abs() - 0.5) * 0.003;
                tilt *= isUnder ? -1.0 : 1.0;
                final value = isUnder ? min(temp.value, pi / 2) : temp.value;
                return Transform(
                  transform: Matrix4.rotationY(value)
                    ..setEntry(3, 0, tilt),
                  alignment: Alignment.center,
                  child: child,
                );
              }
          );
        },
        child: Card.outlined(
          key: ValueKey(index),
          clipBehavior: Clip.antiAlias,
          child: InkWell(
            onTap:(){
              if (index == -1) return;
              setState(() {
                index = 1 - index;
                select = 0;
                san();
              });
            },
            child: Padding(
              padding: const EdgeInsets.all(8.0),
              child: Row(
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  Text(index == 1 ? widget.route.DestinationStopNameZh ?? "error" : widget.route.DepartureStopNameZh ?? "error",style: TextStyle(fontWeight: FontWeight.bold,fontSize: 20)),
                  Icon(Icons.arrow_forward_rounded),
                  Text(index == 1 ? widget.route.DepartureStopNameZh ?? "error" : widget.route.DestinationStopNameZh ?? "error",style: TextStyle(fontWeight: FontWeight.bold,fontSize: 20)),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
  Widget buildStation(dynamic station,bool selected){
    return SizedBox(
        height: 50,
        child: ListTile(
          title: Text(station.StopName?["Zh_tw"] ?? "",style: TextStyle(fontWeight: FontWeight.bold)),
          tileColor: selected ? colorscheme.primaryContainer : colorscheme.surface,
      ),
    );
  }
  Widget buildTime(DateTime time,int S2S){
    DateTime temp = time.add(Duration(seconds: S2S));
    return ListTile(
      title: Text(DateFormat("HH:mm").format(time.toLocal()),style: TextStyle(fontWeight: FontWeight.bold)),
    );
  }
  Map<String,int> offset = {};
  void san(){
    offset.clear();
    final temp = _data["S2S"] ?? [].firstWhereOrNull((element) =>  element["Direction"] == index);
    int sum = 0;
    if(temp != null){
      List<dynamic> temp2 = temp["TravelTimes"][0]["S2STimes"];
      for (var i in temp2){
        sum += (i["RunTime"] as num).toInt();
        offset[i["ToStationID"]] = sum;
      }
    }
  }
  @override
  Widget build(BuildContext context){
    if (loading){
      return Scaffold(
        appBar: AppBar(
          centerTitle: true,
          title: Text(widget.route.SubRouteName, style: TextStyle(fontSize: 30,fontWeight: FontWeight.bold)),
        ),
        body: const Center(child: CircularProgressIndicator()),
      );
    }
    final List<dynamic> estimate = _data["estimates"] ?? [];
    final List<dynamic> stop = _data["stops"] ?? [];
    //final List<dynamic> S2S = _data["S2S"] ?? [];
    final List<dynamic> schedule = _data["schedules"] ?? [];
    final Map<String,dynamic> stopMap ={
      for (var i in estimate) i.StopUID: i
    };
    final Map<String,dynamic> scheduleMap = {};
    for (var i in schedule){
      for (var j in i.Stops){
        scheduleMap[j.StopUID] = j;
      }
    }
    List<dynamic> inbound = stop.firstWhereOrNull((element) => element.Direction == 0)?.Stops ?? [];
    List<dynamic> outbound = stop.firstWhereOrNull((element) => element.Direction == 1)?.Stops ?? [];
    outbound.sort((a,b) => a.StopSequence.compareTo(b.StopSequence));
    inbound.sort((a,b) => a.StopSequence.compareTo(b.StopSequence));
    final display = index == -1 || index == 0 ? inbound : outbound;
    final temp = scheduleMap[display[select].StopUID].TimeTables;
    return DefaultTabController(
      length: 2,
      child: Scaffold(
        appBar: AppBar(
          centerTitle: true,
          title: Text(widget.route.SubRouteName, style: TextStyle(fontSize: 30, fontWeight: FontWeight.bold)),
          bottom: PreferredSize(
            preferredSize: Size.fromHeight(50.0),
            child: Stack(
              alignment: Alignment.bottomCenter,
              children: [
                TabBar(
                  labelColor: colorscheme.primary,
                  tabs: <Widget> [
                    Tab(text: "路線站牌"),
                    widget.route.City != 'Taipei' && widget.route.City != 'NewTaipei' ? Tab(text: "今日班表") : Tab(text: "班表"),
                  ],
                ),
                AnimatedBuilder(animation: _animationController, builder: (context, child) {
                  return LinearProgressIndicator(
                    value: refresh ? null : (1.0-_animationController.value),
                    color: refresh ? colorscheme.secondary : colorscheme.primary,
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
          children: [
            Column(
              children: [
                flip(),
                Expanded(
                  child: ListView.builder(
                    itemCount: display.length,
                    itemBuilder: (context, index) => buildlisttile(display[index],stopMap[display[index].StopUID],colorscheme, index == 0, index == display.length - 1,scheduleMap[display[index].StopUID]),
                  )
                )
              ],
            ),
            Column(
              children: [
                flip(),
                Expanded(
                  child: Padding(
                    padding: const EdgeInsets.all(8.0),
                    child: Card.outlined(
                      elevation: 2,
                      clipBehavior: Clip.antiAlias,
                      child: Row(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children:[
                            Expanded(
                                flex: 5,
                                child: Column(
                                  children: [
                                    Padding(
                                      padding: const EdgeInsets.all(8.0),
                                      child: Text("站牌",style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold)),
                                    ),
                                    Divider(height: 1,thickness: 1.67,color: Colors.grey[400]),
                                    Expanded(
                                      child: Row(
                                          children: [
                                            Expanded(
                                              child: ListView.builder(
                                                controller: ball,
                                                itemCount: display.length,
                                                itemBuilder: (context, index) => buildStation(display[index], index == select),
                                              ),
                                            ),
                                            SizedBox(
                                              width: 30,
                                              height: (display.length-1) * 50+32,
                                              child: RotatedBox(
                                                  quarterTurns: 1,
                                                  child: SliderTheme(
                                                    data: SliderTheme.of(context).copyWith(
                                                      trackHeight: 16,
                                                      overlayShape: SliderComponentShape.noOverlay,
                                                      thumbShape: RoundSliderThumbShape(
                                                        enabledThumbRadius: 12,
                                                        elevation: 4,
                                                      ),
                                                    ),
                                                    child: Slider(
                                                      value: select.toDouble(),
                                                      autofocus: true,
                                                      min: 0,
                                                      max: (display.length - 1).toDouble(),
                                                      divisions: display.length - 1,
                                                      onChanged: (value) {
                                                        setState(() {
                                                          select = value.toInt();
                                                        });
                                                        if(ball.hasClients){
                                                          ball.animateTo(
                                                            ((select*50)-(ball.position.viewportDimension/2)+35) as double,
                                                            duration: Duration(milliseconds: 300),
                                                            curve: Curves.easeInOut,
                                                          );
                                                        }
                                                      },
                                                    ),
                                                  )
                                              ),
                                            ),
                                          ]
                                      ),
                                    )
                                  ],
                                )),
                            VerticalDivider(width: 1,thickness: 1.67,color: Colors.grey[400],indent: 40),
                            Expanded(
                                flex: 4,
                                child: Column(
                                  children: [
                                    Padding(
                                      padding: const EdgeInsets.all(8.0),
                                      child: Text("今日班表",style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold)),
                                    ),
                                    Divider(height: 1,thickness: 1.67,color: Colors.grey[400]),
                                    Expanded(
                                        child: schedule.isEmpty ?
                                        Center(child: Text("今日未營運",style: TextStyle(fontSize: 16,fontWeight: FontWeight.bold))) :
                                        ListView.builder(
                                          itemCount: temp.length,
                                          itemBuilder: (context, index) => buildTime(temp[index].DepartureTime, offset[display[select].StationID] ?? 0),
                                        )
                                    )
                                  ],
                                )
                            ),
                          ],
                        ),
                      ),
                  ),
                ),
              ],
            )
          ],
        ),
      )
    );
  }
}