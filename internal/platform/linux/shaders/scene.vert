#version 450

layout(location = 0) in vec4 inBounds;
layout(location = 1) in vec4 inColor;
layout(location = 2) in vec4 inStyle;
// A triangle's colors at its second and third corner; inColor is the first's.
// The rasterizer interpolates them across the face, so a flat triangle (all
// three equal) is unchanged.
layout(location = 3) in vec4 inShade1;
layout(location = 4) in vec4 inShade2;

layout(push_constant) uniform Frame {
    vec2 viewport;
    vec2 unused;
} frame;

layout(location = 0) out vec4 outColor;
layout(location = 1) out vec2 outLocal;
layout(location = 2) out vec2 outSize;
layout(location = 3) out float outRadius;
layout(location = 4) out vec2 outFragment;
layout(location = 5) out vec2 outTriangleA;
layout(location = 6) out vec2 outTriangleB;
layout(location = 7) out vec2 outTriangleC;
layout(location = 8) flat out uint outTriangleMask;

const vec2 corners[6] = vec2[](
    vec2(0.0, 0.0),
    vec2(1.0, 0.0),
    vec2(0.0, 1.0),
    vec2(0.0, 1.0),
    vec2(1.0, 0.0),
    vec2(1.0, 1.0)
);

void main() {
    vec2 pixel;
    vec2 localPixel;
    vec2 size = inBounds.zw;
    float radius = inStyle.x;
    vec2 triangleA = vec2(0.0);
    vec2 triangleB = vec2(0.0);
    vec2 triangleC = vec2(0.0);
    uint triangleMask = 0;
    vec4 color = inColor;
    if (inStyle.w > 0.5) {
        vec2 points[3] = vec2[](inBounds.xy, inBounds.zw, inStyle.xz);
        pixel = gl_VertexIndex < 3 ? points[gl_VertexIndex] : points[0];
        color = gl_VertexIndex == 1 ? inShade1 : (gl_VertexIndex == 2 ? inShade2 : inColor);
        localPixel = vec2(0.0);
        size = vec2(0.0);
        radius = 0.0;
        triangleA = points[0];
        triangleB = points[1];
        triangleC = points[2];
        triangleMask = uint(inStyle.w + 0.5);
    } else if (inStyle.z > 0.0) {
        // A soft shadow: inBounds and inStyle.x are the rounded rectangle that
        // casts it, inStyle.z its blur, inShade1 = (offset.x, offset.y, spread). The
        // quad grows by the most the shadow reaches, and the local position stays
        // measured from the casting rectangle's corner. Offset, spread and blur
        // reach the fragment shader in the triangle varyings, which a rectangle
        // has no use for.
        float extent = inStyle.z + max(inShade1.z, 0.0) + max(max(inShade1.x, -inShade1.x), max(inShade1.y, -inShade1.y));
        pixel = inBounds.xy - extent + corners[gl_VertexIndex] * (inBounds.zw + 2.0 * extent);
        localPixel = pixel - inBounds.xy;
        triangleA = inShade1.xy;
        triangleB = vec2(inShade1.z, inStyle.z);
    } else {
        pixel = inBounds.xy + corners[gl_VertexIndex] * inBounds.zw;
        localPixel = corners[gl_VertexIndex] * inBounds.zw;
    }
    // Vulkan's positive viewport height maps NDC -1 to the top and +1 to the
    // bottom, unlike Metal and D3D. Keep Viva's logical origin at top-left.
    gl_Position = vec4(pixel / frame.viewport * 2.0 - 1.0, 0.0, 1.0);
    outColor = color;
    outLocal = localPixel;
    outSize = size;
    outRadius = radius;
    outFragment = pixel;
    outTriangleA = triangleA;
    outTriangleB = triangleB;
    outTriangleC = triangleC;
    outTriangleMask = triangleMask;
}
