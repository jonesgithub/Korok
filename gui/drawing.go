package gui

import (
	"github.com/go-gl/mathgl/mgl32"
	"korok.io/korok/engi/math"
	"korok.io/korok/gfx"
	geo "math"
	"korok.io/korok/gfx/bk"
)

type DrawListFlags uint32
const (
	FlagAntiAliasedLine DrawListFlags = iota
	FlagAntiAliasedFill
)
// Rounding corner:
// A: 0x0000 0001 top-left
// B: 0x0000 0002 top-right
// C: 0x0000 0004 down-right
// D: 0x0000 0008 down-left
type FlagCorner uint32

const (
	FlagCornerNone        FlagCorner = 0x0000
	FlagCornerTopLeft                = 0x0001
	FlagCornerTopRight               = 0x0002
	FlagCornerBottomRight            = 0x0004
	FlagCornerBottomLeft             = 0x0008
	FlagCornerAll                    = 0x000F
)

type Align uint32

const (
	AlignCenter Align = iota
	AlignLeft		  = 1 << iota
	AlignRight		  = 1 << iota
	AlignTop 		  = 1 << iota
	AlignBottom		  = 1 << iota
)

// DrawList provide method to write primitives to buffer
type DrawCmd struct {
	ElemCount int
	ClipRect mgl32.Vec4
	TextureId uint16
}

type DrawIdx uint16

type DrawVert struct {
	xy mgl32.Vec2
	uv  mgl32.Vec2
	color uint32
}

type DrawList struct {
	CmdBuffer []DrawCmd
	IdxBuffer []DrawIdx
	VtxBuffer []DrawVert

	cmdIndex, idxIndex, vtxIndex int
	cmdCap, idxCap, vtxCap int

	// Data *DrawListSharedData
	OwnerName string // 窗口名
	VtxCurrentIdx int // VtxBuffer.Size

	// 指向当前正在使用的 cmdbuffer 的位置
	VtxWriter []DrawVert
	IdxWriter []DrawIdx

	ClipRectStack[]mgl32.Vec4
	TextureIdStack []uint16

	// path
	path [64]mgl32.Vec2
	pathUsed int


	FullScreen mgl32.Vec4
	TexUVWhitePixel mgl32.Vec2
	CircleVtx12 [12]mgl32.Vec2
	Font gfx.FontSystem
	FontSize float32

	Flags DrawListFlags
}

func NewDrawList() *DrawList {
	dl := &DrawList{}
	dl.Initialize()
	return dl
}

func (dl *DrawList) Initialize() {
	dl.CmdBuffer = make([]DrawCmd, 1024)
	dl.IdxBuffer = make([]DrawIdx, 2024)
	dl.VtxBuffer = make([]DrawVert, 2024)

	// TODO
	dl.TexUVWhitePixel = mgl32.Vec2{0, 0}

	// TODO bake circle vertex!!
	for i := 0; i < 12; i++ {
		sin := float32(geo.Sin((6.28/12)*(float64(i))))
		cos := float32(geo.Cos((6.28/12)*float64(i)))
		dl.CircleVtx12[i] = mgl32.Vec2{cos, sin}
	}
}

func (dl *DrawList) Empty() bool {
	return dl.vtxIndex == 0 || dl.idxIndex == 0
}

func (dl *DrawList) Size() (idx, vdx int) {
	idx = dl.idxIndex
	vdx = dl.vtxIndex
	return
}

// TODO
func (dl *DrawList) Clear() {
	dl.cmdIndex = 0
	dl.idxIndex = 0
	dl.vtxIndex = 0
}

func (dl *DrawList) PathClear() {
	dl.pathUsed = 0
}

func (dl *DrawList) PathLineTo(pos mgl32.Vec2) {
	if n := len(dl.path); dl.pathUsed < n-1 {
		dl.path[dl.pathUsed] = pos
		dl.pathUsed += 1
	}
}

func (dl *DrawList) PathLineToMergeDuplicate(pos mgl32.Vec2) {
	//if (_Path.Size == 0 || memcmp(&_Path[_Path.Size-1], &pos, 8) != 0)
	//	_Path.push_back(pos);
}

func (dl *DrawList) PathFillConvex(col uint32) {
	dl.AddConvexPolyFilled(dl.path[:dl.pathUsed], col);
	dl.pathUsed = 0
}

// default: thickness=1.0
func (dl *DrawList) PathStroke(color uint32, thickness float32, closed bool)  {
	dl.AddPolyLine(dl.path[:dl.pathUsed], color, thickness, closed)
	dl.PathClear()
}


func (dl *DrawList) CurrentClipRect() (clip mgl32.Vec4) {
	if n := len(dl.ClipRectStack); n > 0 {
		clip = dl.ClipRectStack[n-1]
	} else {
		clip = dl.FullScreen
	}
	return
}

func (dl *DrawList) CurrentTextureId() (id uint16) {
	if n := len(dl.TextureIdStack); n > 0 {
		id = dl.TextureIdStack[n-1]
	}
	return
}

// will result in new draw-call
func (dl *DrawList) UpdateClipRect() {
	//clip := dl.CurrentClipRect()
}

func (dl *DrawList) UpdateTextureId() {

}

// Clip 相关的操作
func (dl *DrawList) PushClipRect(min, max mgl32.Vec2, intersectCurrentClip bool) {
	cr := mgl32.Vec4{min[0], min[1], max[0], max[1]}
	if intersectCurrentClip && len(dl.ClipRectStack) > 0{
		current := dl.ClipRectStack[len(dl.ClipRectStack)-1]
		if cr[0] < current[0] {
			cr[0] = current[0]
		}
		if cr[1] < current[1] {
			cr[1] = current[1]
		}
		if cr[2] > current[2] {
			cr[2] = current[2]
		}
		if cr[3] > current[3] {
			cr[3] = current[3]
		}
		cr[2] = math.Max(cr[0], cr[2])
		cr[3] = math.Max(cr[1], cr[3])

		dl.ClipRectStack = append(dl.ClipRectStack, cr)
		dl.UpdateClipRect()
	}
}

func (dl *DrawList) PushClipRectFullScreen() {
	min := mgl32.Vec2{dl.FullScreen[0], dl.FullScreen[1]}
	max := mgl32.Vec2{dl.FullScreen[2], dl.FullScreen[3]}
	dl.PushClipRect(min, max, false)
}

func (dl *DrawList) PopClipRect() {
	if n := len(dl.ClipRectStack); n > 0 {
		dl.ClipRectStack = dl.ClipRectStack[:n-1]
	}
}

func (dl *DrawList) GetClipRectMin() mgl32.Vec2 {
	return mgl32.Vec2{0, 0 }
}

func (dl *DrawList) GetClipRectMax() mgl32.Vec2 {
	return mgl32.Vec2{0, 0 }
}

func (dl *DrawList) PushTextureId(texId uint16) {
	dl.TextureIdStack = append(dl.TextureIdStack, texId)
}

func (dl *DrawList) PopTextureId() {
	if n := len(dl.TextureIdStack); n > 0 {
		dl.TextureIdStack = dl.TextureIdStack[:n-1]
	}
}

// primitive operation
func (dl *DrawList) PrimReserve(idxCount, vtxCount int) {
	dl.VtxWriter = dl.VtxBuffer[dl.vtxIndex:dl.vtxIndex+vtxCount]
	dl.IdxWriter = dl.IdxBuffer[dl.idxIndex:dl.idxIndex+idxCount]
}

func (dl *DrawList) PrimRect(min, max mgl32.Vec2, color uint32) {
	uv := dl.TexUVWhitePixel
	a, b, c, d := min, mgl32.Vec2{max[0], min[1]}, max, mgl32.Vec2{min[0], max[1]}
	dl.VtxWriter[0] = DrawVert{a, uv, color}
	dl.VtxWriter[1] = DrawVert{b, uv, color}
	dl.VtxWriter[2] = DrawVert{c, uv, color}
	dl.VtxWriter[3] = DrawVert{d, uv, color}

	dl.IdxWriter[0] = DrawIdx(dl.vtxIndex+0)
	dl.IdxWriter[1] = DrawIdx(dl.vtxIndex+1)
	dl.IdxWriter[2] = DrawIdx(dl.vtxIndex+2)

	dl.IdxWriter[3] = DrawIdx(dl.vtxIndex+0)
	dl.IdxWriter[4] = DrawIdx(dl.vtxIndex+2)
	dl.IdxWriter[5] = DrawIdx(dl.vtxIndex+3)

	dl.vtxIndex += 4
	dl.idxIndex += 6
}

func (dl *DrawList) PrimRectUV(a, c mgl32.Vec2, uva, uvc mgl32.Vec2, color uint32) {
	b, d := mgl32.Vec2{c[0], a[1]}, mgl32.Vec2{a[0], c[1]}
	uvb, uvd := mgl32.Vec2{uvc[0], uva[1]}, mgl32.Vec2{uva[0], uvc[1]}

	dl.VtxWriter[0] = DrawVert{a, uva, color}
	dl.VtxWriter[1] = DrawVert{b, uvb, color}
	dl.VtxWriter[2] = DrawVert{c, uvc, color}
	dl.VtxWriter[3] = DrawVert{d, uvd, color}

	ii := dl.vtxIndex
	dl.IdxWriter[0] = DrawIdx(ii+0)
	dl.IdxWriter[1] = DrawIdx(ii+1)
	dl.IdxWriter[2] = DrawIdx(ii+2)
	dl.IdxWriter[3] = DrawIdx(ii+0)
	dl.IdxWriter[4] = DrawIdx(ii+2)
	dl.IdxWriter[5] = DrawIdx(ii+3)

	dl.idxIndex += 6
	dl.vtxIndex += 4
}

func (dl *DrawList) PrimQuadUV(a, b, c, d mgl32.Vec2, uva, uvb,uvc, uvd mgl32.Vec2, color uint32) {
	// vertex
	dl.VtxWriter[0] = DrawVert{a, uva, color}
	dl.VtxWriter[1] = DrawVert{b, uvb, color}
	dl.VtxWriter[2] = DrawVert{c, uvc, color}
	dl.VtxWriter[3] = DrawVert{d, uvd, color}

	ii := dl.vtxIndex
	dl.IdxWriter[0] = DrawIdx(ii+0)
	dl.IdxWriter[1] = DrawIdx(ii+1)
	dl.IdxWriter[2] = DrawIdx(ii+2)
	dl.IdxWriter[3] = DrawIdx(ii+0)
	dl.IdxWriter[4] = DrawIdx(ii+2)
	dl.IdxWriter[5] = DrawIdx(ii+3)

	dl.vtxIndex += 4
	dl.idxIndex += 6
}

// 此处生成最终的顶点数据和索引数据
// 当前并不支持抗锯齿！！简单的用顶点生成线段
func (dl *DrawList) AddPolyLine(points []mgl32.Vec2, color uint32, thickness float32, closed bool) {
	pointsCount := len(points)
	if pointsCount < 2 {
		return
	}
	uv := dl.TexUVWhitePixel
	count := pointsCount
	if !closed {
		count = pointsCount - 1
	}
	// Non Anti-aliased Stroke
	idxCount := count * 6
	vtxCount := count * 4
	dl.PrimReserve(idxCount, vtxCount)

	for i1 := 0; i1 < count; i1 ++{
		i2 := i1 + 1
		if i2 == pointsCount {
			i2 = 0
		}
		p1, p2 := points[i1], points[i2]

		diff := p2.Sub(p1)

		invLength := InvLength(diff, 1.0)
		diff = diff.Mul(invLength)
		dx := diff[0] * (thickness * 0.5)
		dy := diff[1] * (thickness * 0.5)

		vi := i1*4
		dl.VtxWriter[vi+0] = DrawVert{mgl32.Vec2{p1[0]+dy, p1[1]-dx}, uv, color}
		dl.VtxWriter[vi+1] = DrawVert{mgl32.Vec2{p2[0]+dy, p2[1]-dx}, uv, color}
		dl.VtxWriter[vi+2] = DrawVert{mgl32.Vec2{p2[0]-dy, p2[1]+dx}, uv, color}
		dl.VtxWriter[vi+3] = DrawVert{mgl32.Vec2{p1[0]-dy, p1[1]+dx}, uv, color}

		ii := i1*6
		dl.IdxWriter[ii+0] = DrawIdx(dl.vtxIndex+0)
		dl.IdxWriter[ii+1] = DrawIdx(dl.vtxIndex+1)
		dl.IdxWriter[ii+2] = DrawIdx(dl.vtxIndex+2)
		dl.IdxWriter[ii+3] = DrawIdx(dl.vtxIndex+0)
		dl.IdxWriter[ii+4] = DrawIdx(dl.vtxIndex+2)
		dl.IdxWriter[ii+5] = DrawIdx(dl.vtxIndex+3)

		dl.vtxIndex += 4
		dl.idxIndex += 6
	}
	dl.AddCommand(idxCount)
}

// Non Anti-aliased Fill
func (dl *DrawList) AddConvexPolyFilled(points []mgl32.Vec2, color uint32) {
	uv := dl.TexUVWhitePixel
	pointCount := len(points)

	idxCount := (pointCount-2)*3
	vtxCount := pointCount
	dl.PrimReserve(idxCount, vtxCount)

	for i := 0; i < vtxCount; i++ {
		dl.VtxWriter[i] = DrawVert{points[i], uv, color}
	}
	for i, ii := 2, 0; i < pointCount; i, ii = i+1, ii+3 {
		dl.IdxWriter[ii+0] = DrawIdx(dl.vtxIndex+0)
		dl.IdxWriter[ii+1] = DrawIdx(dl.vtxIndex+i-1)
		dl.IdxWriter[ii+2] = DrawIdx(dl.vtxIndex+i)
	}

	dl.vtxIndex += vtxCount
	dl.idxIndex += idxCount
	dl.AddCommand(idxCount)
}

// 此处圆角的算法：
// 使用一个12边形近似圆形，采用中心放射算法，计算出
// 各个角度的sin/cos, 然后通过公式，得到圆圆形顶点
// f(x) = centre.x + cos()*radius
// f(y) = centre.y + sin()*radius
// 以上, 可以提前算好 sin/cos 加速整个过程
func (dl *DrawList) PathArcToFast(centre mgl32.Vec2, radius float32, min12, max12 int) {
	if radius == 0 || min12 > max12 {
		dl.path[dl.pathUsed] = centre; dl.pathUsed ++
		return
	}
	for a := min12; a <= max12; a++ {
		x := centre[0] + dl.CircleVtx12[a%12][0] * radius
		y := centre[1] + dl.CircleVtx12[a%12][1] * radius
		dl.path[dl.pathUsed] = mgl32.Vec2{x, y}
		dl.pathUsed ++
	}
}

func (dl *DrawList) PathArcTo(centre mgl32.Vec2, radius float32, min, max float32, segments int) {
	if radius == 0 {
		dl.path[dl.pathUsed] = centre; dl.pathUsed++
		return
	}
	for i := 0; i <= segments; i++ {
		a := float64(min + (float32(i)/float32(segments)) * (max-min))
		x := centre[0] + float32(geo.Cos(a)) * radius
		y := centre[1] + float32(geo.Sin(a)) * radius
		dl.path[dl.pathUsed] = mgl32.Vec2{x, y}
		dl.pathUsed ++
	}

}

func (dl *DrawList) PathBezierCurveTo(p2, p3, p4 mgl32.Vec2, segments int) {

}

func (dl *DrawList) PathRect(a, b mgl32.Vec2, rounding float32, corners FlagCorner) {
	if rounding <= 0 || corners == FlagCornerNone {
		dl.PathLineTo(a)
		dl.PathLineTo(mgl32.Vec2{b[0], a[1]})
		dl.PathLineTo(b)
		dl.PathLineTo(mgl32.Vec2{a[0], b[1]})
	} else {
		var bl, br, tr, tl float32
		if (corners & FlagCornerBottomLeft) != 0 {
			bl = rounding
		}
		if (corners & FlagCornerBottomRight) != 0 {
			br = rounding
		}
		if (corners & FlagCornerTopRight) != 0 {
			tr = rounding
		}
		if (corners & FlagCornerTopLeft) != 0 {
			tl = rounding
		}
		dl.PathArcToFast(mgl32.Vec2{a[0]+bl, a[1]+bl}, bl, 6, 9) // bottom-left
		dl.PathArcToFast(mgl32.Vec2{b[0]-br, a[1]+br}, br, 9, 12)// bottom-right
		dl.PathArcToFast(mgl32.Vec2{b[0]-tr, b[1]-tr}, tr, 0, 3) // top-right
		dl.PathArcToFast(mgl32.Vec2{a[0]+tl, b[1]-tl}, tl, 3, 6) // top-left
	}
}

func (dl *DrawList) AddLine(a, b mgl32.Vec2, color uint32, thickness float32) {
	dl.PathLineTo(a.Add(mgl32.Vec2{.5, .5}))
	dl.PathLineTo(b.Add(mgl32.Vec2{.5, .5}))
	dl.PathStroke(color, thickness, false)
}

// 所有非填充图形看来都是使用路径实现的
func (dl *DrawList) AddRect(a, b mgl32.Vec2, color uint32, rounding float32, roundFlags FlagCorner, thickness float32) {
	//dl.PathRect(a.Add(mgl32.Vec2{5, .5}), b.Sub(mgl32.Vec2{.5, .5}), rounding, roundFlags)
	// TODO
	dl.PathRect(a, b, rounding, roundFlags)
	dl.PathStroke(color, thickness, true)
}

func (dl *DrawList) AddRectFilled(min, max mgl32.Vec2, color uint32, rounding float32, corner FlagCorner) {
	if rounding > 0 && corner != FlagCornerNone {
		dl.PathRect(min, max, rounding, corner)
		dl.PathFillConvex(color)
	} else {
		dl.PrimReserve(6, 4)
		dl.PrimRect(min, max, color)
		dl.AddCommand(6)
	}
}

func (dl *DrawList) AddRectFilledMultiColor() {

}

func (dl *DrawList) AddQuad(a, b, c, d mgl32.Vec2, color uint32, thickness float32) {
	dl.PathLineTo(a)
	dl.PathLineTo(b)
	dl.PathLineTo(c)
	dl.PathLineTo(d)
	dl.PathStroke(color, thickness, true)
}

func (dl *DrawList) AddQuadFilled(a, b, c, d mgl32.Vec2, color uint32) {
	dl.PathLineTo(a)
	dl.PathLineTo(b)
	dl.PathLineTo(c)
	dl.PathLineTo(d)
	dl.PathFillConvex(color)
}

func (dl *DrawList) AddTriangle(a, b, c mgl32.Vec2, color uint32, thickness float32) {
	dl.PathLineTo(a)
	dl.PathLineTo(b)
	dl.PathLineTo(c)
	dl.PathStroke(color, thickness, true)
}

func (dl *DrawList) AddTriangleFilled(a, b, c mgl32.Vec2, color uint32) {
	dl.PathLineTo(a)
	dl.PathLineTo(b)
	dl.PathLineTo(c)
	dl.PathFillConvex(color)
}

func (dl *DrawList) AddCircle(centre mgl32.Vec2, radius float32, color uint32, segments int, thickness float32) {
	max := PI * 2 * float32(segments-1)/float32(segments)
	dl.PathArcTo(centre, radius, 0.0, max, segments)
	dl.PathStroke(color, thickness, true)
}

func (dl *DrawList) AddCircleFilled(centre mgl32.Vec2, radius float32, color uint32, segments int) {
	max := PI * 2 * float32(segments-1)/float32(segments)
	dl.PathArcTo(centre, radius,0.0, max, segments)
	dl.PathFillConvex(color)
}

func (dl *DrawList) AddBezierCurve(pos0 mgl32.Vec2, cp0, cp1 mgl32.Vec2, pos1 mgl32.Vec2,
	color uint32, thickness float32, segments int) {
	dl.PathLineTo(pos0)
	dl.PathBezierCurveTo(cp0, cp1, pos1, segments)
	dl.PathStroke(color, thickness, false)
}

func (dl *DrawList) AddImage(texId uint16, a, b mgl32.Vec2, uva, uvb mgl32.Vec2, color uint32) {
	if n := len(dl.TextureIdStack); n == 0 || texId != dl.TextureIdStack[n-1]  {
		dl.PushTextureId(texId)
		defer dl.PopTextureId()
	}

	dl.PrimReserve(6, 4)
	dl.PrimRectUV(a, b, uva, uvb, color)
	dl.AddCommand(6)
}

func (dl *DrawList) AddImageQuad(texId uint16, a, b, c, d mgl32.Vec2, uva, uvb, uvc, uvd mgl32.Vec2, color uint32) {
	if n := len(dl.TextureIdStack); n == 0 || texId != dl.TextureIdStack[n-1] {
		dl.PushTextureId(texId)
		defer dl.PopTextureId()
	}
	dl.PrimReserve(6, 4)
	dl.PrimQuadUV(a, b, c, d, uva, uvb, uvc, uvd, color)
	dl.AddCommand(6)
}

func (dl *DrawList) AddImageRound(texId uint16, a, b mgl32.Vec2, uva, uvb mgl32.Vec2, color uint32, rounding float32, corners FlagCorner) {
	if rounding <= 0 || (corners & FlagCornerAll) == 0 {
		dl.AddImage(texId, a, b, uva, uvb, color)
		return
	}
	if n := len(dl.TextureIdStack); n == 0 || texId != dl.TextureIdStack[n-1] {
		dl.PushTextureId(texId)
		defer dl.PopTextureId()
	}

	dl.PathRect(a, b, rounding, corners)
	dl.PathFillConvex(color)

	// map uv to vertex - linear scale
	xySize, uvSize := b.Sub(a), uvb.Sub(uva)
	var scale mgl32.Vec2
	if xySize[0] != 0 {
		scale[0] = uvSize[0]/xySize[0]
	}
	if xySize[1] != 0 {
		scale[1] = uvSize[1]/xySize[1]
	}

	// clamp??
	for i  := range dl.VtxWriter {
		vertex := &dl.VtxWriter[i]
		dx := (vertex.xy[0] - a[0]) * scale[0]
		dy := (vertex.xy[1] - a[1]) * scale[1]
		vertex.uv = mgl32.Vec2{uva[0]+dx, uva[1]+dy}
	}
}

// NinePatch Algorithm
//  12   13   14   15
//       x1   x2     max
//  +----+----+----+
//  |    |    |    |
//  |    |    |p1  |
//  +----+----+----+ y2
//  |    |    |    |
//  |    |p0  |    |
//  +----+----+----+ y1
//  |    |    |    |
//  |    |    |    |
//  +----+----+----+
//min
//  0    1    2    3
//patch = {x1, x2, y1, y2} % TextureSize
func (dl *DrawList) AddImageNinePatch(texId uint16, min, max mgl32.Vec2, uva, uvb mgl32.Vec2, patch mgl32.Vec4, color uint32) {
	if n := len(dl.TextureIdStack); n == 0 || texId != dl.TextureIdStack[n-1]  {
		dl.PushTextureId(texId)
		defer dl.PopTextureId()
	}

	_, tex := bk.R.Texture(texId)
	texSize := mgl32.Vec2{tex.Width, tex.Height}

	idxCount, vtxCount := 9 * 6, 16
	dl.PrimReserve(idxCount, vtxCount)

	x1, x2, y1, y2 := min[0]+patch[0]*texSize[0], max[0]-patch[1]*texSize[0], min[1]+patch[2]*texSize[1], max[1]-patch[3]*texSize[1]
	uvw, uvh := uvb[0]-uva[0], uvb[1]-uva[1]
	u1, u2, v1, v2 := uva[0]+patch[0]*uvw, uvb[0]-patch[1]*uvw, uva[1]+patch[2]*uvh, uvb[1]-patch[3]*uvh

	if x2 < x1 {
		x1 = (min[0] + max[0])/2; x2 = x1
	}
	if y2 < y1 {
		y1 = (min[1] + max[1])/2; y2 = y1
	}

	vtxWriter := dl.VtxWriter
	idxWriter := dl.IdxWriter

	// fill vertex
	vtxWriter[0] = DrawVert{min, uva, color}
	vtxWriter[1] = DrawVert{mgl32.Vec2{x1, min[1]}, mgl32.Vec2{u1, uva[1]}, color}
	vtxWriter[2] = DrawVert{mgl32.Vec2{x2, min[1]}, mgl32.Vec2{u2, uva[1]}, color}
	vtxWriter[3] = DrawVert{mgl32.Vec2{max[0], min[1]}, mgl32.Vec2{uvb[0], uva[1]}, color}

	vtxWriter[4] = DrawVert{mgl32.Vec2{min[0], y1}, mgl32.Vec2{uva[0], v1}, color}
	vtxWriter[5] = DrawVert{mgl32.Vec2{x1, y1}, mgl32.Vec2{u1, v1}, color}
	vtxWriter[6] = DrawVert{mgl32.Vec2{x2, y1}, mgl32.Vec2{u2, v1}, color}
	vtxWriter[7] = DrawVert{mgl32.Vec2{max[0], y1}, mgl32.Vec2{uvb[0], v1}, color}

	vtxWriter[8] = DrawVert{mgl32.Vec2{min[0], y2}, mgl32.Vec2{uva[0], v2}, color}
	vtxWriter[9] = DrawVert{mgl32.Vec2{x1, y2}, mgl32.Vec2{u1, v2}, color}
	vtxWriter[10] = DrawVert{mgl32.Vec2{x2, y2}, mgl32.Vec2{u2, v2}, color}
	vtxWriter[11] = DrawVert{mgl32.Vec2{max[0], y2}, mgl32.Vec2{uvb[0], v2}, color}

	vtxWriter[12] = DrawVert{mgl32.Vec2{min[0], max[1]}, mgl32.Vec2{uva[0], uvb[1]}, color}
	vtxWriter[13] = DrawVert{mgl32.Vec2{x1, max[1]}, mgl32.Vec2{u1, uvb[1]}, color}
	vtxWriter[14] = DrawVert{mgl32.Vec2{x2, max[1]}, mgl32.Vec2{u2, uvb[1]}, color}
	vtxWriter[15] = DrawVert{max, uvb, color}

	// fill index
	ii := uint16(dl.vtxIndex)
	for i, v := range ninePatchIndex {
		idxWriter[i] = DrawIdx(ii+v)
	}
	dl.idxIndex += idxCount
	dl.vtxIndex += vtxCount

	dl.AddCommand(idxCount)
}

var ninePatchIndex = [54]uint16 {
	0, 1, 5,  0, 5,  4,   1, 2,  6,  1, 6,  5,   2,  3, 7,  2,  7,  6,
	4, 5, 9,  4, 9,  8,   5, 6,  10, 5, 10, 9,   6,  7, 11, 6,  11, 10,
	8, 9, 13, 8, 13, 12,  9, 10, 14, 9, 14, 13,  10, 11,15, 10, 15, 14,
}

func (dl *DrawList) AddText(pos mgl32.Vec2, text string, font gfx.FontSystem, fontSize float32, color uint32, wrapWidth float32) (size mgl32.Vec2){
	if text == "" {
		return
	}
	if font == nil {
		font = dl.Font
	}
	if fontSize == 0 {
		fontSize = dl.FontSize
	}

	fr := &FontRender{
		DrawList:dl,
		fontSize:fontSize,
		font:font,
		color:color,
	}

	if wrapWidth > 0 {
		size = fr.RenderWrapped(pos, text, wrapWidth)
	} else {
		size = fr.RenderText1(pos, text)
	}
	return
}

// 每次绘制都会产生一个 Command （可能会造成内存浪费! 1k cmd = 1000 * 6 * 4 = 24k）
// 为了减少内存可以一边添加一边尝试向前合并
func (dl *DrawList) AddCommand(elemCount int) {
	clip := dl.CurrentClipRect()
	tex  := dl.CurrentTextureId()

	if ii := dl.cmdIndex; ii == 0 {
		dl.CmdBuffer[ii] = DrawCmd{elemCount, clip, tex}
		dl.cmdIndex += 1
	} else {
		if prev  := &dl.CmdBuffer[ii-1]; prev.ClipRect == clip && prev.TextureId == tex {
			prev.ElemCount += elemCount
		} else {
			dl.CmdBuffer[ii] = DrawCmd{elemCount,clip,tex}
			dl.cmdIndex += 1
		}
	}
}

func (dl *DrawList) Commands() []DrawCmd {
	return dl.CmdBuffer[:dl.cmdIndex]
}



