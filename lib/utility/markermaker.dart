import 'dart:ui' as ui;
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_svg/flutter_svg.dart';
import 'package:google_maps_flutter/google_maps_flutter.dart';

class Markermaker {
  Future<BitmapDescriptor> make(BuildContext context,String i) async {
    double pixel = 100 * MediaQuery.of(context).devicePixelRatio;
    final SvgAssetLoader loader = SvgAssetLoader(i);
    final PictureInfo pictureInfo = await vg.loadPicture(loader,null);
    final ui.PictureRecorder recorder = ui.PictureRecorder();
    final ui.Canvas canvas = ui.Canvas(recorder);
    final double scale = pixel / pictureInfo.size.width;
    canvas.scale(scale, scale);
    canvas.drawPicture(pictureInfo.picture);
    final ui.Image image = await recorder.endRecording().toImage(pixel.toInt(), ((pictureInfo.size.height / pictureInfo.size.width)*pixel).toInt());
    final ByteData? byteData = await image.toByteData(format: ui.ImageByteFormat.png);
    pictureInfo.picture.dispose();
    return BitmapDescriptor.fromBytes(byteData!.buffer.asUint8List());
  }
  Future<Map<String,BitmapDescriptor>> loadicons(BuildContext context) async{
    Map<String,BitmapDescriptor> icons = {};
    List<String> paths = [ "1","AIR","AM","Bike","BL","BR","Bus","C","G","K","O1","O2","R1","R2","V","Y","Pegman"];
    for(String i in paths) {
      icons[i] = await make(context, "assets/icons/$i.svg");
    }
    return icons;
  }
}