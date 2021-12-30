# forked from dxvk @ ecd7b67069a6e2a4b78a8a929fd287030d283ccb (2021-12-17)

project('northstar-dedicated-d3d11', ['c', 'cpp'], meson_version : '>= 0.46')

if (target_machine.cpu_family() != 'x86_64')
  error('only x86_64 supported')
endif

add_project_arguments('-DNOMINMAX', language : 'cpp')

x_compiler = meson.get_compiler('cpp')
x_is_msvc = x_compiler.get_id() == 'msvc'
x_cpp_std='c++17'

if x_is_msvc
  add_project_arguments('/std:' + x_cpp_std, language : 'cpp')
endif

if x_compiler.get_id() == 'clang'
  if x_compiler.has_argument('-Wno-unused-private-field')
    add_project_arguments('-Wno-unused-private-field', language: 'cpp')
  endif
  if x_compiler.has_argument('-Wno-microsoft-exception-spec')
    add_project_arguments('-Wno-microsoft-exception-spec', language: 'cpp')
  endif
endif

if not x_is_msvc
  add_project_arguments('-D_WIN32_WINNT=0xa00', language : 'cpp')

  # We need to set the section alignment for debug symbols to
  # work properly as well as avoiding a memcpy from the Wine loader.
  if x_compiler.has_link_argument('-Wl,--file-alignment=4096')
    add_global_link_arguments('-Wl,--file-alignment=4096', language: 'cpp')
  endif

  # Wine's built-in back traces only work with dwarf2 symbols
  if get_option('debug') and target_machine.system() == 'windows'
    if x_compiler.has_argument('-gstrict-dwarf') and x_compiler.has_argument('-gdwarf-2')
      add_project_arguments('-gstrict-dwarf', '-gdwarf-2', language: ['c', 'cpp'])
    endif
  endif
endif

if x_is_msvc
  wrc = find_program('rc')
else
  add_global_link_arguments('-static', '-static-libgcc', language: 'c')
  add_global_link_arguments('-static', '-static-libgcc', '-static-libstdc++', language: 'cpp')

  wrc = find_program('windres')
endif

if x_is_msvc
  res_ext = '.res'
else
  res_ext = '.o'
endif

if x_is_msvc
  wrc_generator = generator(wrc,
  output    : [ '@BASENAME@' + res_ext ],
  arguments : [ '/fo', '@OUTPUT@', '@INPUT@' ])
else
  wrc_generator = generator(wrc,
  output    : [ '@BASENAME@' + res_ext ],
  arguments : [ '-i', '@INPUT@', '-o', '@OUTPUT@' ])
endif

d3d11_res = wrc_generator.process('version.rc')

util_src = [
  'com_guid.cc',
  'com_private_data.cc',
  'log.cc',
  'util_string.cc',
]

dxgi_src = [
  'dxgi_adapter.cc',
  'dxgi_factory.cc',
  'dxgi_output.cc',
]

d3d11_src = [
  'd3d11_annotation.cc',
  'd3d11_blend.cc',
  'd3d11_buffer.cc',
  'd3d11_class_linkage.cc',
  'd3d11_cmdlist.cc',
  'd3d11_context.cc',
  'd3d11_context_def.cc',
  'd3d11_context_imm.cc',
  'd3d11_depth_stencil.cc',
  'd3d11_device.cc',
  'd3d11_input_layout.cc',
  'd3d11_main.cc',
  'd3d11_query.cc',
  'd3d11_rasterizer.cc',
  'd3d11_resource.cc',
  'd3d11_sampler.cc',
  'd3d11_state.cc',
  'd3d11_state_object.cc',
  'd3d11_swapchain.cc',
  'd3d11_texture.cc',
  'd3d11_view_dsv.cc',
  'd3d11_view_rtv.cc',
  'd3d11_view_srv.cc',
  'd3d11_view_uav.cc',
]

shared_library('d3d11', util_src + dxgi_src + d3d11_src, d3d11_res,
  name_prefix         : '',
  install             : true,
  vs_module_defs      : 'd3d11.def',
  override_options    : ['cpp_std=c++17'],
)
