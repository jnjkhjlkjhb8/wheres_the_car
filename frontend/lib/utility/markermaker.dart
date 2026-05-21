import 'dart:ui' as ui;
import 'package:flutter/services.dart';
import 'package:flutter_svg/flutter_svg.dart';
import 'package:google_maps_flutter/google_maps_flutter.dart';

class Markermaker {
  Future<BitmapDescriptor> make(String i,double pixel) async {
    final SvgAssetLoader loader = SvgAssetLoader(i);
    final PictureInfo pictureInfo = await vg.loadPicture(loader,null);
    final ui.PictureRecorder recorder = ui.PictureRecorder();
    final ui.Canvas canvas = ui.Canvas(recorder);
    var size = 10;
    canvas.scale(size / pictureInfo.size.width, size / pictureInfo.size.width);
    canvas.drawPicture(pictureInfo.picture);
    final ui.Image image = await recorder.endRecording().toImage(size,size);
    final ByteData? byteData = await image.toByteData(format: ui.ImageByteFormat.png);
    pictureInfo.picture.dispose();
    return BitmapDescriptor.bytes(byteData!.buffer.asUint8List());
  }
  Future<Map<String,BitmapDescriptor>> loadicons() async{
    final double temp = ui.PlatformDispatcher.instance.views.first.devicePixelRatio;
    Map<String,BitmapDescriptor> icons = {};
    List<String> paths = [ "1","AIR","AM","Bike","BL","BR","Bus","C","G","K","O1","O2","R1","R2","V","Y","Pegman","BD","RD","GD"];
    for(String i in paths) {
      icons[i] = await make("assets/marker/$i.svg",temp);
    }
    return icons;
  }
}