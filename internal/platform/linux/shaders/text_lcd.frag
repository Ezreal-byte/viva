#version 450

// LCD text: the label texture holds the coverage of each color channel (red, green and blue)
// and the target is blended per channel with dual-source blending. outColor is what is added
// to the target, outMask the factor, per channel, by which the target is kept:
//   target = outColor + target * (1 - outMask).

layout(set = 0, binding = 0) uniform sampler2D labelTexture;

layout(push_constant) uniform Tint {
    layout(offset = 32) vec4 color;
} tint;

layout(location = 0) in vec2 inUV;
layout(location = 0, index = 0) out vec4 outColor;
layout(location = 0, index = 1) out vec4 outMask;

void main() {
    vec4 texel = texture(labelTexture, inUV);
    vec3 coverage = texel.rgb * tint.color.a;
    outColor = vec4(tint.color.rgb * coverage, texel.a * tint.color.a);
    outMask = vec4(coverage, texel.a * tint.color.a);
}
