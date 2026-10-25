package ui

// This file triangulates closed rings (one outer ring and any holes) with the
// earcut algorithm: holes are joined to the outer ring through a bridge chosen
// by casting a ray from the hole's leftmost vertex, ears are clipped from the
// linked ring, and when clipping stalls the ring is first filtered, then its
// small self-intersections are cured, and as a last resort it is split along a
// valid diagonal. It follows Mapbox's earcut (ISC license), without the
// z-order hash that only pays off for very large rings, and works on caller
// scratch so a steady-state paint allocates nothing.
//
// Earcut knows nothing of anti-aliasing. Which triangle edges lie on the
// outline (and so are feathered) is worked out afterwards from the input rings:
// an edge is an outline edge when it joins two neighboring points of one ring.
// Bridges and diagonals are not, so shared edges stay opaque.

// earNode is one vertex of the circular linked ring earcut cuts apart.
type earNode struct {
	x, y       float64
	index      int32 // position in the flat point list, shared by split copies
	prev, next int32 // node ids, -1 for none
}

// earcutScratch is the reusable working memory of one triangulation.
type earcutScratch struct {
	points    []Point   // the cleaned rings, one after another
	ends      []int32   // exclusive end of each ring in points
	nodes     []earNode // the linked rings
	triangles []int32   // output: three point indexes per triangle
	holes     []int32   // leftmost node of each hole, sorted by x
}

// earClipRings triangulates canvas.ringPoints as one fill group: ring zero is
// the outer shape and the remaining rings are holes. Consecutive duplicate
// points and rings of fewer than three points are dropped first.
func (canvas *Canvas) earClipRings(color Color) {
	scratch := &canvas.earcut
	scratch.points = scratch.points[:0]
	scratch.ends = scratch.ends[:0]
	start := int32(0)
	for _, end := range canvas.ringEnds {
		ring := canvas.ringPoints[start:end]
		start = end
		mark := len(scratch.points)
		for _, point := range ring {
			if len(scratch.points) > mark && distance(scratch.points[len(scratch.points)-1], point) < 1e-4 {
				continue
			}
			scratch.points = append(scratch.points, point)
		}
		for len(scratch.points)-mark > 1 && distance(scratch.points[mark], scratch.points[len(scratch.points)-1]) < 1e-4 {
			scratch.points = scratch.points[:len(scratch.points)-1]
		}
		if len(scratch.points)-mark < 3 {
			scratch.points = scratch.points[:mark]
			if len(scratch.ends) == 0 {
				return // no outer ring, nothing to fill
			}
			continue // a hole with no area
		}
		scratch.ends = append(scratch.ends, int32(len(scratch.points)))
	}
	if len(scratch.ends) == 0 {
		return
	}

	scratch.nodes = scratch.nodes[:0]
	scratch.triangles = scratch.triangles[:0]
	outer := scratch.linkedList(scratch.points[:scratch.ends[0]], 0, true)
	if outer < 0 || scratch.nodes[outer].next == scratch.nodes[outer].prev {
		return
	}
	if len(scratch.ends) > 1 {
		outer = scratch.eliminateHoles(outer)
	}
	scratch.earcutLinked(outer, 0)

	for at := 0; at+2 < len(scratch.triangles); at += 3 {
		a, b, c := scratch.triangles[at], scratch.triangles[at+1], scratch.triangles[at+2]
		mask := uint8(0)
		if scratch.outlineEdge(a, b) {
			mask |= 1 << 0
		}
		if scratch.outlineEdge(b, c) {
			mask |= 1 << 1
		}
		if scratch.outlineEdge(c, a) {
			mask |= 1 << 2
		}
		canvas.appendTriangleEdges(scratch.points[a], scratch.points[b], scratch.points[c], color, mask)
	}
}

// outlineEdge reports whether points i and j are neighbors in one input ring.
func (scratch *earcutScratch) outlineEdge(i, j int32) bool {
	start := int32(0)
	for _, end := range scratch.ends {
		if i >= start && i < end {
			if j < start || j >= end {
				return false
			}
			step := j - i
			return step == 1 || step == -1 || step == end-start-1 || step == -(end-start-1)
		}
		start = end
	}
	return false
}

// linkedList links points into a circular list wound clockwise or not as asked
// and returns its last node, or -1 for an empty ring.
func (scratch *earcutScratch) linkedList(points []Point, base int32, clockwise bool) int32 {
	sum := 0.0
	for i, j := 0, len(points)-1; i < len(points); j, i = i, i+1 {
		sum += (float64(points[j].X) - float64(points[i].X)) * (float64(points[i].Y) + float64(points[j].Y))
	}
	last := int32(-1)
	if clockwise == (sum > 0) {
		for i := 0; i < len(points); i++ {
			last = scratch.insertNode(base+int32(i), points[i], last)
		}
	} else {
		for i := len(points) - 1; i >= 0; i-- {
			last = scratch.insertNode(base+int32(i), points[i], last)
		}
	}
	if last >= 0 && scratch.equals(last, scratch.nodes[last].next) {
		scratch.removeNode(last)
		last = scratch.nodes[last].next
	}
	return last
}

func (scratch *earcutScratch) insertNode(index int32, point Point, last int32) int32 {
	id := int32(len(scratch.nodes))
	node := earNode{x: float64(point.X), y: float64(point.Y), index: index, prev: id, next: id}
	if last >= 0 {
		node.next = scratch.nodes[last].next
		node.prev = last
		scratch.nodes[scratch.nodes[last].next].prev = id
		scratch.nodes[last].next = id
	}
	scratch.nodes = append(scratch.nodes, node)
	return id
}

func (scratch *earcutScratch) removeNode(id int32) {
	node := scratch.nodes[id]
	scratch.nodes[node.next].prev = node.prev
	scratch.nodes[node.prev].next = node.next
}

func (scratch *earcutScratch) equals(a, b int32) bool {
	return scratch.nodes[a].x == scratch.nodes[b].x && scratch.nodes[a].y == scratch.nodes[b].y
}

// filterPoints removes nodes that repeat their neighbor or lie on the line
// between their neighbors, and returns the new end of the stretch.
func (scratch *earcutScratch) filterPoints(start, end int32) int32 {
	if start < 0 {
		return start
	}
	if end < 0 {
		end = start
	}
	p := start
	for {
		again := false
		node := scratch.nodes[p]
		if scratch.equals(p, node.next) || scratch.area(node.prev, p, node.next) == 0 {
			scratch.removeNode(p)
			p = node.prev
			end = p
			if p == scratch.nodes[p].next {
				break
			}
			again = true
		} else {
			p = node.next
		}
		if !again && p == end {
			break
		}
	}
	return end
}

// eliminateHoles joins every hole to the outer ring, leftmost hole first.
func (scratch *earcutScratch) eliminateHoles(outer int32) int32 {
	scratch.holes = scratch.holes[:0]
	for ring := 1; ring < len(scratch.ends); ring++ {
		start := scratch.ends[ring-1]
		list := scratch.linkedList(scratch.points[start:scratch.ends[ring]], start, false)
		if list < 0 {
			continue
		}
		scratch.holes = append(scratch.holes, scratch.leftmost(list))
	}
	// Insertion sort: there are few holes, and a closure would allocate.
	for i := 1; i < len(scratch.holes); i++ {
		for j := i; j > 0 && compareEarNodes(scratch.nodes[scratch.holes[j]], scratch.nodes[scratch.holes[j-1]]) < 0; j-- {
			scratch.holes[j], scratch.holes[j-1] = scratch.holes[j-1], scratch.holes[j]
		}
	}
	for _, hole := range scratch.holes {
		outer = scratch.eliminateHole(hole, outer)
	}
	return outer
}

func compareEarNodes(a, b earNode) int {
	switch {
	case a.x < b.x:
		return -1
	case a.x > b.x:
		return 1
	case a.y < b.y:
		return -1
	case a.y > b.y:
		return 1
	}
	return 0
}

func (scratch *earcutScratch) leftmost(start int32) int32 {
	best := start
	p := start
	for {
		if scratch.nodes[p].x < scratch.nodes[best].x || scratch.nodes[p].x == scratch.nodes[best].x && scratch.nodes[p].y < scratch.nodes[best].y {
			best = p
		}
		p = scratch.nodes[p].next
		if p == start {
			return best
		}
	}
}

// eliminateHole splices one hole into the outer ring along a bridge.
func (scratch *earcutScratch) eliminateHole(hole, outer int32) int32 {
	bridge := scratch.findHoleBridge(hole, outer)
	if bridge < 0 {
		return outer
	}
	reverse := scratch.splitPolygon(bridge, hole)
	scratch.filterPoints(reverse, scratch.nodes[reverse].next)
	return scratch.filterPoints(bridge, scratch.nodes[bridge].next)
}

// findHoleBridge finds the outer-ring vertex a hole's leftmost vertex can see:
// cast a ray to the left, take the nearer end of the first edge it hits, then
// prefer a reflex vertex inside the triangle that makes with the ray.
func (scratch *earcutScratch) findHoleBridge(hole, outer int32) int32 {
	nodes := scratch.nodes
	p := outer
	hx, hy := nodes[hole].x, nodes[hole].y
	qx := -1e300
	m := int32(-1)
	for {
		next := nodes[p].next
		if scratch.equals(hole, p) {
			return p
		}
		if hy <= nodes[p].y && hy >= nodes[next].y && nodes[next].y != nodes[p].y {
			x := nodes[p].x + (hy-nodes[p].y)*(nodes[next].x-nodes[p].x)/(nodes[next].y-nodes[p].y)
			if x <= hx && x > qx {
				qx = x
				if nodes[p].x < nodes[next].x {
					m = p
				} else {
					m = next
				}
				if x == hx {
					return m // the hole touches this edge: its left end is the bridge
				}
			}
		}
		p = next
		if p == outer {
			break
		}
	}
	if m < 0 {
		return -1
	}

	stop := m
	mx, my := nodes[m].x, nodes[m].y
	tanMin := 1e300
	p = m
	for {
		px, py := nodes[p].x, nodes[p].y
		if hx >= px && px >= mx && hx != px {
			ax, bx := qx, hx
			if hy < my {
				ax, bx = hx, qx
			}
			if pointInTriangleXY(ax, hy, mx, my, bx, hy, px, py) {
				tan := abs64(hy-py) / (hx - px)
				if scratch.locallyInside(p, hole) && (tan < tanMin || tan == tanMin && (px > nodes[m].x || px == nodes[m].x && scratch.sectorContainsSector(m, p))) {
					m = p
					tanMin = tan
				}
			}
		}
		p = nodes[p].next
		if p == stop {
			break
		}
	}
	return m
}

// sectorContainsSector reports whether the sector at m contains the one at p.
func (scratch *earcutScratch) sectorContainsSector(m, p int32) bool {
	return scratch.area(scratch.nodes[m].prev, m, scratch.nodes[p].prev) < 0 && scratch.area(scratch.nodes[p].next, m, scratch.nodes[m].next) < 0
}

// earcutLinked clips ears off the ring one by one. pass escalates when a whole
// lap finds none: 0 filters the ring, 1 cures local self-intersections, 2
// splits the ring in two.
func (scratch *earcutScratch) earcutLinked(ear int32, pass int) {
	if ear < 0 {
		return
	}
	stop := ear
	for scratch.nodes[ear].prev != scratch.nodes[ear].next {
		prev, next := scratch.nodes[ear].prev, scratch.nodes[ear].next
		if scratch.isEar(ear) {
			scratch.triangles = append(scratch.triangles, scratch.nodes[prev].index, scratch.nodes[ear].index, scratch.nodes[next].index)
			scratch.removeNode(ear)
			// Skipping the next vertex avoids long slivers.
			ear = scratch.nodes[next].next
			stop = ear
			continue
		}
		ear = next
		if ear == stop {
			switch pass {
			case 0:
				scratch.earcutLinked(scratch.filterPoints(ear, -1), 1)
			case 1:
				ear = scratch.cureLocalIntersections(scratch.filterPoints(ear, -1))
				scratch.earcutLinked(ear, 2)
			case 2:
				scratch.splitEarcut(ear)
			}
			break
		}
	}
}

// isEar reports whether the vertex's triangle is convex and holds no other
// vertex of the ring.
func (scratch *earcutScratch) isEar(ear int32) bool {
	nodes := scratch.nodes
	b := nodes[ear]
	a, c := nodes[b.prev], nodes[b.next]
	if scratch.area(b.prev, ear, b.next) >= 0 {
		return false // reflex: not an ear
	}
	x0, x1 := min(a.x, b.x, c.x), max(a.x, b.x, c.x)
	y0, y1 := min(a.y, b.y, c.y), max(a.y, b.y, c.y)
	for p := c.next; p != b.prev; p = nodes[p].next {
		node := nodes[p]
		if node.x >= x0 && node.x <= x1 && node.y >= y0 && node.y <= y1 &&
			!(a.x == node.x && a.y == node.y) && pointInTriangleXY(a.x, a.y, b.x, b.y, c.x, c.y, node.x, node.y) &&
			scratch.area(node.prev, p, node.next) >= 0 {
			return false
		}
	}
	return true
}

// cureLocalIntersections removes small self-intersections: where the edge
// before a vertex crosses the edge after its successor, the triangle between
// them is emitted and both vertices dropped.
func (scratch *earcutScratch) cureLocalIntersections(start int32) int32 {
	p := start
	for {
		a := scratch.nodes[p].prev
		b := scratch.nodes[scratch.nodes[p].next].next
		next := scratch.nodes[p].next
		if !scratch.equals(a, b) && scratch.intersects(a, p, next, b) && scratch.locallyInside(a, b) && scratch.locallyInside(b, a) {
			scratch.triangles = append(scratch.triangles, scratch.nodes[a].index, scratch.nodes[p].index, scratch.nodes[b].index)
			scratch.removeNode(p)
			scratch.removeNode(next)
			p = b
			start = b
		}
		p = scratch.nodes[p].next
		if p == start {
			break
		}
	}
	return scratch.filterPoints(p, -1)
}

// splitEarcut splits the ring along a valid diagonal and triangulates each
// half.
func (scratch *earcutScratch) splitEarcut(start int32) {
	a := start
	for {
		b := scratch.nodes[scratch.nodes[a].next].next
		for b != scratch.nodes[a].prev {
			if scratch.nodes[a].index != scratch.nodes[b].index && scratch.isValidDiagonal(a, b) {
				c := scratch.splitPolygon(a, b)
				a = scratch.filterPoints(a, scratch.nodes[a].next)
				c = scratch.filterPoints(c, scratch.nodes[c].next)
				scratch.earcutLinked(a, 0)
				scratch.earcutLinked(c, 0)
				return
			}
			b = scratch.nodes[b].next
		}
		a = scratch.nodes[a].next
		if a == start {
			return
		}
	}
}

func (scratch *earcutScratch) isValidDiagonal(a, b int32) bool {
	nodes := scratch.nodes
	return nodes[nodes[a].next].index != nodes[b].index && nodes[nodes[a].prev].index != nodes[b].index &&
		!scratch.intersectsPolygon(a, b) &&
		(scratch.locallyInside(a, b) && scratch.locallyInside(b, a) && scratch.middleInside(a, b) &&
			(scratch.area(nodes[a].prev, a, nodes[b].prev) != 0 || scratch.area(a, nodes[b].prev, b) != 0) ||
			scratch.equals(a, b) && scratch.area(nodes[a].prev, a, nodes[a].next) > 0 && scratch.area(nodes[b].prev, b, nodes[b].next) > 0)
}

// area is twice the signed area of the triangle p, q, r.
func (scratch *earcutScratch) area(p, q, r int32) float64 {
	nodes := scratch.nodes
	return (nodes[q].y-nodes[p].y)*(nodes[r].x-nodes[q].x) - (nodes[q].x-nodes[p].x)*(nodes[r].y-nodes[q].y)
}

func pointInTriangleXY(ax, ay, bx, by, cx, cy, px, py float64) bool {
	return (cx-px)*(ay-py) >= (ax-px)*(cy-py) && (ax-px)*(by-py) >= (bx-px)*(ay-py) && (bx-px)*(cy-py) >= (cx-px)*(by-py)
}

func sign(value float64) int {
	switch {
	case value > 0:
		return 1
	case value < 0:
		return -1
	}
	return 0
}

// intersects reports whether segments p1q1 and p2q2 intersect.
func (scratch *earcutScratch) intersects(p1, q1, p2, q2 int32) bool {
	o1 := sign(scratch.area(p1, q1, p2))
	o2 := sign(scratch.area(p1, q1, q2))
	o3 := sign(scratch.area(p2, q2, p1))
	o4 := sign(scratch.area(p2, q2, q1))
	if o1 != o2 && o3 != o4 {
		return true
	}
	return o1 == 0 && scratch.onSegment(p1, p2, q1) || o2 == 0 && scratch.onSegment(p1, q2, q1) ||
		o3 == 0 && scratch.onSegment(p2, p1, q2) || o4 == 0 && scratch.onSegment(p2, q1, q2)
}

// onSegment reports whether q lies within the bounding box of segment pr
// (call it only when the three are collinear).
func (scratch *earcutScratch) onSegment(p, q, r int32) bool {
	nodes := scratch.nodes
	return nodes[q].x <= max(nodes[p].x, nodes[r].x) && nodes[q].x >= min(nodes[p].x, nodes[r].x) &&
		nodes[q].y <= max(nodes[p].y, nodes[r].y) && nodes[q].y >= min(nodes[p].y, nodes[r].y)
}

// intersectsPolygon reports whether the diagonal ab crosses an edge of the ring.
func (scratch *earcutScratch) intersectsPolygon(a, b int32) bool {
	nodes := scratch.nodes
	p := a
	for {
		next := nodes[p].next
		if nodes[p].index != nodes[a].index && nodes[next].index != nodes[a].index &&
			nodes[p].index != nodes[b].index && nodes[next].index != nodes[b].index && scratch.intersects(p, next, a, b) {
			return true
		}
		p = next
		if p == a {
			return false
		}
	}
}

// locallyInside reports whether the diagonal from a to b starts into the
// polygon's interior at a.
func (scratch *earcutScratch) locallyInside(a, b int32) bool {
	nodes := scratch.nodes
	if scratch.area(nodes[a].prev, a, nodes[a].next) < 0 {
		return scratch.area(a, b, nodes[a].next) >= 0 && scratch.area(a, nodes[a].prev, b) >= 0
	}
	return scratch.area(a, b, nodes[a].prev) < 0 || scratch.area(a, nodes[a].next, b) < 0
}

// middleInside reports whether the middle of the diagonal ab is inside the
// polygon.
func (scratch *earcutScratch) middleInside(a, b int32) bool {
	nodes := scratch.nodes
	inside := false
	px, py := (nodes[a].x+nodes[b].x)/2, (nodes[a].y+nodes[b].y)/2
	p := a
	for {
		next := nodes[p].next
		if (nodes[p].y > py) != (nodes[next].y > py) && nodes[next].y != nodes[p].y &&
			px < (nodes[next].x-nodes[p].x)*(py-nodes[p].y)/(nodes[next].y-nodes[p].y)+nodes[p].x {
			inside = !inside
		}
		p = next
		if p == a {
			return inside
		}
	}
}

// splitPolygon joins a and b with a bridge, producing two rings, and returns
// the second ring's node on b's side.
func (scratch *earcutScratch) splitPolygon(a, b int32) int32 {
	a2 := int32(len(scratch.nodes))
	scratch.nodes = append(scratch.nodes, earNode{x: scratch.nodes[a].x, y: scratch.nodes[a].y, index: scratch.nodes[a].index})
	b2 := int32(len(scratch.nodes))
	scratch.nodes = append(scratch.nodes, earNode{x: scratch.nodes[b].x, y: scratch.nodes[b].y, index: scratch.nodes[b].index})
	nodes := scratch.nodes
	an, bp := nodes[a].next, nodes[b].prev
	nodes[a].next = b
	nodes[b].prev = a
	nodes[a2].next = an
	nodes[an].prev = a2
	nodes[b2].next = a2
	nodes[a2].prev = b2
	nodes[bp].next = b2
	nodes[b2].prev = bp
	return b2
}
